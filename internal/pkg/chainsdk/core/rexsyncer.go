package chain

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

//var rex_syncer_old_log = logging.Logger("rsyncer")

//var REQ_BLOCKS_PER_REQUEST = int32(10) // ask for n blocks per request
//var TASK_DURATION_ADJ = 5 * 1000       // in millseconds
//var MAXIMUM_TASK_DURATION = 60 * 1000  // is millseconds
//var MINIMUM_TASK_DURATION = 5 * 10000  // is millseconds

//type SyncResult struct {
//	TaskId uint64
//	Data   interface{}
//}
//
//type SyncerStatus uint
//
//const (
//	RUNNING SyncerStatus = iota
//	IDLE
//	CLOSED
//)

type SyncTask struct {
	TaskId       uint64 //blockId
	ReqBlockNum  int32
	taskDuration int
	TaskCtx      context.Context
	TaskCancel   context.CancelFunc
	chSyncResult chan *SyncResult
	results      []*quorumpb.ReqBlockResp
	taskDone     chan bool
}

type RexSyncer struct {
	GroupItem *quorumpb.GroupItem
	GroupId   string
	nodename  string
	chain     *Chain
	cdnIface  def.ChainDataSyncIface
	chainCtx  context.Context

	rexSyncerCtx    context.Context
	rexSyncerCancel context.CancelFunc

	currTask   *SyncTask
	chSyncTask chan *SyncTask

	Status              SyncerStatus
	currContinueFailCnt uint
	currDelay           int

	LastSyncResult *def.RexSyncResult
	lock           sync.Mutex
}

func NewRexSyncer(chainCtx context.Context, grpItem *quorumpb.GroupItem, nodename string, cdnIface def.ChainDataSyncIface, chain *Chain) *RexSyncer {
	rex_syncer_log.Debugf("<%s> NewRexSyncer called", grpItem.GroupId)
	rs := &RexSyncer{}
	rs.GroupItem = grpItem
	rs.GroupId = grpItem.GroupId
	rs.nodename = nodename
	rs.chain = chain
	rs.cdnIface = cdnIface
	rs.chainCtx = chainCtx
	rs.currTask = nil
	rs.chSyncTask = nil
	rs.Status = IDLE
	rs.currContinueFailCnt = 0
	rs.currDelay = 0
	rs.LastSyncResult = nil

	return rs
}

func (rs *RexSyncer) Start() {
	rex_syncer_log.Debugf("<%s> Start called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.rexSyncerCtx != nil {
		rs.rexSyncerCancel()
	}
	rs.Status = RUNNING

	rs.chSyncTask = make(chan *SyncTask)
	rs.rexSyncerCtx, rs.rexSyncerCancel = context.WithCancel(rs.chainCtx)

	//start sync worker
	go rs.SyncWorker(rs.chainCtx, rs.chSyncTask)

	//start sync task generator
	go func() {
		for {
			select {
			case <-rs.chainCtx.Done():
				rex_syncer_log.Debugf("<%s> RexSyncer exit", rs.GroupId)
				return
			case <-rs.rexSyncerCtx.Done():
				rex_syncer_log.Debugf("<%s> RexSyncer exit", rs.GroupId)
				return
			default:
				if rs.Status == CLOSED {
					rex_syncer_log.Debugf("<%s> RexSyncer exit", rs.GroupId)
					return
				}
				if rs.rexSyncerCtx.Err() != nil {
					rex_syncer_log.Debugf("<%s> RexSyncer exit", rs.GroupId)
					return
				}
				if rs.chainCtx.Err() != nil {
					rex_syncer_log.Debugf("<%s> RexSyncer exit", rs.GroupId)
					return
				}

				//get next task
				fmt.Println("===========ok, getNextSyncTask")
				newTask := rs.getNextSyncTask()
				rs.chSyncTask <- newTask
				<-newTask.taskDone
			}
		}
	}()
}

func (rs *RexSyncer) Stop() {
	rex_syncer_log.Debugf("<%s> Stop called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.Status = CLOSED

	if rs.rexSyncerCancel != nil {
		rs.rexSyncerCancel()
	}

	//close(rs.chSyncTask)
}

func (rs *RexSyncer) SyncWorker(chainCtx context.Context, chSyncTask <-chan *SyncTask) {
	rex_syncer_log.Debugf("<%s> SyncWorker called", rs.GroupId)
	for {
		select {
		case <-chainCtx.Done():
			rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
			return

		case <-rs.rexSyncerCtx.Done():
			rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
			if rs.currTask != nil {
				rex_syncer_log.Debugf("<%s> SyncWorker cancel current task", rs.GroupId)
				rs.currTask.TaskCancel()
				rs.currTask = nil
			}
			return

		case task, beforeClosed := <-chSyncTask:
			if !beforeClosed {
				rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
				return
			}

			if rs.Status == CLOSED {
				rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
				return
			}
			if chainCtx.Err() != nil {
				rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
				return
			}
			if rs.rexSyncerCtx.Err() != nil {
				rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
				return
			}

			rex_syncer_log.Debugf("<%s> SyncWorker set current task, taskId <%d>, duration <%d>", rs.GroupId, task.TaskId, task.taskDuration)
			rs.currTask = task

			var trx *quorumpb.Trx
			var trxerr error

			trx, trxerr = rs.chain.GetTrxFactory().GetReqBlocksTrx("", rs.GroupId, task.TaskId, task.ReqBlockNum)
			if trxerr != nil {
				rex_syncer_log.Warningf("<%s> SyncWorker run task get trx failed, err <%s>", rs.GroupId, trxerr.Error())
			}

			connMgr, err := conn.GetConn().GetConnMgr(rs.GroupId)
			if err != nil {
				rex_syncer_log.Warningf("<%s> SyncWorker run task get connMgr failed, err <%s>", rs.GroupId, err.Error())
			}
			blockBundles := &quorumpb.BlocksBundle{}
			err = fmt.Errorf("FOR TEST, disable SendReq. try DSCache")
			block, err := rs.chain.GetBlockFromDSCache(rs.GroupId, task.TaskId, rs.nodename)
			for block != nil {
				blockBundles.Blocks = append(blockBundles.Blocks, block)
				block, err = rs.chain.GetBlockFromDSCache(rs.GroupId, block.BlockId+1, rs.nodename)
			}

			if len(blockBundles.Blocks) > 0 { // blocks find from cache, apply blocks
				rs.chain.ApplyBlocks(blockBundles.Blocks)
				//update last sync result
				//TODO: update sync info, set pubkey as myself
				//rs.LastSyncResult = &def.RexSyncResult{
				//	Provider:              winnerResp.ProviderPubkey,
				//	FromBlock:             winnerResp.FromBlock,
				//	BlockProvided:         winnerResp.BlksProvided,
				//	SyncResult:            winnerResp.Result.String(),
				//	LastSyncTaskTimestamp: time.Now().Unix(),
				//}

				//reset continue fail cnt
				rs.currContinueFailCnt = 0
				task.taskDone <- true
			} else {
				fmt.Println("=======sync from network...")
				// err = connMgr.SendReqTrxRex(trx)
			}

			_ = trx
			_ = connMgr
			if err != nil {
				rex_syncer_log.Warningf("<%s> SyncWorker run task sendReq failed , err <%s>", rs.GroupId, err.Error())
			}

		TASKDONE:
			for {
				select {
				case <-task.TaskCtx.Done():
					rex_syncer_log.Debugf("<%s> SyncWorker run task, ctx done exit", rs.GroupId)
					return

				case result := <-task.chSyncResult:
					rex_syncer_log.Debugf("<%s> SyncWorker run task get result", rs.GroupId)
					//check if the resp is what we are waiting for
					if result.TaskId != task.TaskId {
						chain_log.Warningf("<%s> SyncWorker run task get result but id mismatch, requested <%d>, response <%d>", rs.GroupId, task.TaskId, result.TaskId)
					} else {

						reqBlockResp := result.Data.(*quorumpb.ReqBlockResp)
						rex_syncer_log.Debugf("- Receive valid reqBlockResp, provider <%s> result <%s> from block <%d> total <%d> blocks provided",
							reqBlockResp.ProviderPubkey,
							reqBlockResp.Result.String(),
							reqBlockResp.FromBlock,
							len(reqBlockResp.Blocks.Blocks))

						//add valid result to list
						task.results = append(rs.currTask.results, reqBlockResp)
					}
				case <-time.After(time.Duration(task.taskDuration) * time.Millisecond):
					rex_syncer_log.Debugf("<%s> SyncWorker run task done, taskId <%d>", rs.GroupId, task.TaskId)

					//no result found, timeout
					if len(task.results) == 0 {
						rex_syncer_log.Debugf("<%s> SyncWorker run task timeout, no result", rs.GroupId)
						rs.currContinueFailCnt += 1
					} else {
						//select a "winner" response
						//1. choose resp provided the most blocks
						//2. if same, choose response from producers
						rex_syncer_log.Debugf("<%s> SyncWorker run task select winner", rs.GroupId)
						var winnerResp *quorumpb.ReqBlockResp
						for _, resp := range task.results {
							if winnerResp == nil {
								winnerResp = resp
								continue
							}

							if len(resp.Blocks.Blocks) > len(winnerResp.Blocks.Blocks) {
								winnerResp = resp
								continue
							}

							if len(resp.Blocks.Blocks) == len(winnerResp.Blocks.Blocks) {
								if rs.chain.IsProducerByPubkey(resp.ProviderPubkey) {
									winnerResp = resp
								}
							}
						}

						rex_syncer_log.Debugf("<%s> SyncWorker run task winner is <%s>", rs.GroupId, winnerResp.ProviderPubkey)

						switch winnerResp.Result {
						case quorumpb.ReqBlkResult_BLOCK_NOT_FOUND:
							rs.currDelay = MAXIMUM_TASK_DURATION
						case quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP:
							rs.currDelay = MAXIMUM_TASK_DURATION
							rs.chain.ApplyBlocks(winnerResp.Blocks.Blocks)
						case quorumpb.ReqBlkResult_BLOCK_IN_RESP:
							rs.currDelay = 0
							rs.chain.ApplyBlocks(winnerResp.Blocks.Blocks)
						default:
						}

						//update last sync result
						rs.LastSyncResult = &def.RexSyncResult{
							Provider:              winnerResp.ProviderPubkey,
							FromBlock:             winnerResp.FromBlock,
							BlockProvided:         winnerResp.BlksProvided,
							SyncResult:            winnerResp.Result.String(),
							LastSyncTaskTimestamp: time.Now().Unix(),
						}

						//reset continue fail cnt
						rs.currContinueFailCnt = 0
					}
					task.taskDone <- true
					break TASKDONE
				}
			}
			rex_syncer_log.Debugf("<%s> SyncWorker run task done, taskId <%d>, duration <%d>", rs.GroupId, task.TaskId, task.taskDuration)
		}
	}
}

func (rs *RexSyncer) GetCurrentTask() *SyncTask {
	rex_syncer_log.Debugf("<%s> GetCurrentTask called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	return rs.currTask
}

func (rs *RexSyncer) GetSyncerStatus() SyncerStatus {
	rex_syncer_log.Debugf("<%s> GetSyncerStatus called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	return rs.Status
}

func (rs *RexSyncer) GetLastRexSyncResult() (*def.RexSyncResult, error) {
	rex_syncer_log.Debugf("<%s> GetLastRexSyncResult called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	if rs.LastSyncResult == nil {
		return nil, fmt.Errorf("no valid rex sync result yet")
	}

	return rs.LastSyncResult, nil
}

func (rs *RexSyncer) GetCurrentTaskDurationAdj() int {
	rex_syncer_log.Debugf("<%s> GetCurrentDelay called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	currAdj := int(rs.currContinueFailCnt)*TASK_DURATION_ADJ + rs.currDelay

	if currAdj > MAXIMUM_TASK_DURATION {
		currAdj = MAXIMUM_TASK_DURATION
	}

	return currAdj
}

func (rs *RexSyncer) AddResult(result *SyncResult) {
	rex_syncer_log.Debugf("<%s> AddResult called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.Status == CLOSED {
		rex_syncer_log.Debugf("<%s> AddResult called, but syncer is closed", rs.GroupId)
		return
	}

	if rs.currTask != nil {
		rs.currTask.chSyncResult <- result
	}
}

// task generators
func (rs *RexSyncer) getNextSyncTask() *SyncTask {
	rex_syncer_log.Debugf("<%s> getNextSyncTask called", rs.GroupId)
	nextBlock := rs.cdnIface.GetCurrBlockId() + uint64(1)
	randDelay := rand.Intn(500)
	taskDuration := int(rs.currContinueFailCnt)*TASK_DURATION_ADJ + rs.currDelay + randDelay
	if taskDuration > MAXIMUM_TASK_DURATION {
		taskDuration = MAXIMUM_TASK_DURATION
	}

	task := &SyncTask{TaskId: nextBlock, ReqBlockNum: REQ_BLOCKS_PER_REQUEST, taskDuration: taskDuration}
	task.TaskCtx, task.TaskCancel = context.WithCancel(rs.chainCtx)
	task.chSyncResult = make(chan *SyncResult, 1)
	task.taskDone = make(chan bool)

	rex_syncer_log.Debugf("<%s> getNextSyncTask, taskId <%d>, taskDuration <%d>", rs.GroupId, task.TaskId, task.taskDuration)
	return task
}

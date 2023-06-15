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

	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
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
	TaskId      uint64 //block
	ReqBlockNum int32
	DelayTime   int
	TriggerTime int64
}

type RexSyncer struct {
	GroupId  string
	nodename string
	cdnIface def.ChainDataSyncIface
	chainCtx *Chain

	//chan signals
	taskq   chan *SyncTask
	resultq chan *SyncResult

	Status            SyncerStatus
	mustatus          sync.RWMutex
	CurrRetryCount    uint
	CurrentDely       int
	CurrentTask       *SyncTask
	CurrentTaskCancel context.CancelFunc

	LastSyncResult *def.RexSyncResult
}

func NewRexSyncer(groupid string, nodename string, cdnIface def.ChainDataSyncIface, chainCtx *Chain) *RexSyncer {
	rex_syncer_log.Debugf("<%s> NewRexSyncer called", groupid)

	rs := &RexSyncer{}
	rs.GroupId = groupid
	rs.nodename = nodename
	rs.cdnIface = cdnIface
	rs.chainCtx = chainCtx

	rex_syncer_log.Debugf("<%s> Init rex syncer channels", rs.GroupId)
	rs.taskq = make(chan *SyncTask)
	rs.resultq = make(chan *SyncResult)

	rs.Status = IDLE
	rs.CurrentTask = nil
	rs.CurrentDely = 0

	rs.LastSyncResult = nil
	return rs
}

func (rs *RexSyncer) GetCurrentTask() *SyncTask {
	return rs.CurrentTask
}

func (rs *RexSyncer) GetSyncerStatus() SyncerStatus {
	return rs.Status
}

func (rs *RexSyncer) Start() {
	rex_syncer_log.Debugf("<%s> Start called", rs.GroupId)

	//start taskq
	go func() {
		for task := range rs.taskq {
			if rs.Status == CLOSED {
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
			//calculate current delay
			task.DelayTime += int(rs.CurrRetryCount)*SYNC_BLOCK_FREQ_ADJ + rs.CurrentDely
			if task.DelayTime > MAXIMUM_DELAY_DURATION {
				task.DelayTime = MAXIMUM_DELAY_DURATION
			}

			task.TriggerTime = time.Now().Unix() + int64(task.DelayTime)/1000
			taskTimeout := task.DelayTime + SYNC_BLOCK_TASK_TIMEOUT
			rex_syncer_log.Debugf("<%s> get task <%d> from taskq, set task timeout to <%d>", rs.GroupId, task.TaskId, taskTimeout)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(taskTimeout)*time.Millisecond)
			rs.runTask(ctx, task, cancel)
		}
	}()

	//start resultq
	go func() {
		for result := range rs.resultq {
			if rs.Status == CLOSED {
				return
			}
			rs.handleResult(result)
		}
	}()

	task := rs.newSyncBlockTask()
	rs.AddTask(task)
}

func (rs *RexSyncer) Stop() {
	rex_syncer_log.Debugf("<%s> Stop called", rs.GroupId)
	rs.mustatus.Lock()
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
	if rs.LastSyncResult == nil {
		return nil, fmt.Errorf("no valid rex sync result yet")
	}

	if rs.CurrentTask != nil {
		rs.LastSyncResult.NextSyncTaskTimeStamp = rs.CurrentTask.TriggerTime
	} else {
		return nil, fmt.Errorf("try again")
	}

	return rs.LastSyncResult, nil
}

func safeClose(ch chan struct{}) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}

func safeCloseTaskQ(ch chan *SyncTask) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}

func safeCloseResultQ(ch chan *SyncResult) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}

func (rs *RexSyncer) runTask(ctx context.Context, task *SyncTask, cancel context.CancelFunc) error {
	//TODO: close this goroutine when the processTask func return. add some defer signal?
	rex_syncer_log.Debugf("runTask called, taskId <%d>, retry <%d>", task.TaskId, rs.CurrRetryCount)
	go func() {
		rs.CurrentTask = task //set current task
		rs.CurrentTaskCancel = cancel
		err := rs.syncBlockTaskSender(task)
		if err != nil {
			rex_syncer_log.Debugf("todo add retry task <%d>", task.TaskId)
			//retry
		}
	}()

	select {
	//case <-rs.taskdone:
	//	return nil
	case <-ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded:
			if rs.Status != CLOSED {
				//a workround, should cancel the ctx for current task
				if rs.CurrentTask != nil {
					rex_syncer_log.Debugf("task <%d> timeout", task.TaskId)
					rs.CurrRetryCount += 1
					rex_syncer_log.Debugf("CurrRetryCount <%d>", rs.CurrRetryCount)

					//remove current task
					rs.CurrentTask = nil
					rs.CurrentTaskCancel = nil
					rs.Status = IDLE

					//start next round with highest epoch
					task := rs.newSyncBlockTask()
					rs.AddTask(task)
				}
			}
		case context.Canceled:
			rex_syncer_log.Debugf("task <%d> done", task.TaskId)
		}
		return nil
	}
}

func (rs *RexSyncer) AddTask(task *SyncTask) {
	rex_syncer_log.Debugf("Gsyncer addTask called")
	go func() {
		rs.mustatus.Lock()
		defer rs.mustatus.Unlock()
		if rs.Status != CLOSED {
			rs.taskq <- task
		}
	}()
}

func (rs *RexSyncer) AddResult(result *SyncResult) {
	go func() {
		rs.mustatus.Lock()
		defer rs.mustatus.Unlock()
		if rs.Status != CLOSED {
			rs.resultq <- result
		}
	}()
}

// task generators
func (rs *RexSyncer) newSyncBlockTask() *SyncTask {
	rex_syncer_log.Debugf("<%s> newSyncBlockTask called", rs.GroupId)
	nextBlock := rs.cdnIface.GetCurrBlockId() + uint64(1)
	randDelay := rand.Intn(500)
	return &SyncTask{TaskId: nextBlock, ReqBlockNum: REQ_BLOCKS_PER_REQUEST, DelayTime: randDelay}
}

func (rs *RexSyncer) syncBlockTaskSender(task *SyncTask) error {
	rex_syncer_log.Debugf("<%s> syncBlockTaskSender called", rs.GroupId)

	var trx *quorumpb.Trx
	var trxerr error

	trx, trxerr = rs.chainCtx.GetTrxFactory().GetReqBlocksTrx("", rs.GroupId, task.TaskId, task.ReqBlockNum)
	if trxerr != nil {
		return trxerr
	}

	connMgr, err := conn.GetConn().GetConnMgr(rs.GroupId)
	if err != nil {
		return err
	}

	rex_syncer_log.Debugf("<%s> sleep <%d> millseconds before send the req", rs.GroupId, task.DelayTime)
	time.Sleep(time.Duration(task.DelayTime) * time.Millisecond)

	//set status to SYNCING since the syncing task is always running and the "real" sync work (after send out reqBlock) only start after sleep
	rs.Status = SYNCING
	return connMgr.SendReqTrxRex(trx)
}

func (rs *RexSyncer) handleResult(result *SyncResult) error {
	rex_syncer_log.Debugf("<%s> handleResult called", rs.GroupId)
	//check if the resp is what we are waiting for
	if rs.CurrentTask == nil {
		rex_syncer_log.Debugf("<%s> CurrentTask is nil, ignore", rs.GroupId)
		return rumerrors.ErrTaskIdMismatch
	}
	if result.TaskId != rs.CurrentTask.TaskId {
		//chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", sr.groupId, rumerrors.ErrEpochMismatch)
		return rumerrors.ErrTaskIdMismatch
	}

	reqBlockResp := result.Data.(*quorumpb.ReqBlockResp)

	rex_syncer_log.Debugf("- Receive valid reqBlockResp, provider <%s> result <%s> from block <%d> total <%d> blocks provided",
		reqBlockResp.ProviderPubkey,
		reqBlockResp.Result.String(),
		reqBlockResp.FromBlock,
		len(reqBlockResp.Blocks.Blocks))

	//Since a valid response is retrieved, finish current task
	/*
		only 1 producer (owner) is supported in this version
		node should only accept BLOCK_NOT_FOUND from group owner and ignore all other BLOCK_NOT_FOUND msg
		TBD, stop only when received BLOCK_NOT_FOUND from F + 1 producers, otherwise continue sync
	*/

	//check if resp is from owner
	isOwner := rs.chainCtx.isOwnerByPubkey(reqBlockResp.ProviderPubkey)

	switch reqBlockResp.Result {
	case quorumpb.ReqBlkResult_BLOCK_NOT_FOUND:
		if isOwner {
			rs.CurrentDely = MAXIMUM_DELAY_DURATION
			chain_log.Debugf("<%s> receive BLOCK_NOT_FOUND from group owner, set delay to <%d>", rs.GroupId, rs.CurrentDely)
		}

	case quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP:
		rs.chainCtx.ApplyBlocks(reqBlockResp.Blocks.Blocks)
		if isOwner {
			rs.CurrentDely = MAXIMUM_DELAY_DURATION
			chain_log.Debugf("<%s> receive BLOCK_IN_RESP_ON_TOP from group owner, apply blocks, set task delay to <%d>", rs.GroupId, rs.CurrentDely)
		}

	case quorumpb.ReqBlkResult_BLOCK_IN_RESP:
		rs.CurrentDely = 0
		chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP from node <%s>, apply all blocks and reset syncer timer to <%d>", rs.GroupId, reqBlockResp.ProviderPubkey, rs.CurrentDely)
		rs.chainCtx.ApplyBlocks(reqBlockResp.Blocks.Blocks)
	default:

	}

	//received something, reset current retry count
	rs.CurrRetryCount = 0

	rs.LastSyncResult = &def.RexSyncResult{
		Provider:              reqBlockResp.ProviderPubkey,
		FromBlock:             reqBlockResp.FromBlock,
		BlockProvided:         reqBlockResp.BlksProvided,
		SyncResult:            reqBlockResp.Result.String(),
		LastSyncTaskTimestamp: time.Now().Unix(),
		NextSyncTaskTimeStamp: -1,
	}

	//rs.taskdone <- struct{}{}
	if rs.CurrentTaskCancel != nil {
		rs.CurrentTaskCancel()
	}
	rs.CurrentTaskCancel = nil
	rs.CurrentTask = nil
	rs.Status = IDLE

	//start next round
	task := rs.newSyncBlockTask()
	rs.AddTask(task)

	return nil
}

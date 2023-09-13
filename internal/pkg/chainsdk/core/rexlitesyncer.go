package chain

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"go.uber.org/atomic"

	rumchaindata "github.com/rumsystem/quorum/pkg/data"
)

var rex_syncer_log = logging.Logger("rsyncer")
var REQ_BLOCKS_PER_REQUEST = int32(10) // ask for n blocks per request
var TASK_DURATION_ADJ = 5              // in seconds
var MAXIMUM_TASK_DURATION = 60         // is seconds
var MINIMUM_TASK_DURATION = 5          // is seconds

type SyncResult struct {
	TaskId uint64
	Data   interface{}
}

type SyncerStatus uint

const (
	RUNNING SyncerStatus = iota
	IDLE
	CLOSED
)

type RexLiteSyncer struct {
	GroupItem          *quorumpb.GroupItem
	GroupId            string
	nodename           string
	chain              *Chain
	cdnIface           def.ChainDataSyncIface
	chainCtx           context.Context
	isResultCollecting atomic.Bool

	rexSyncerCtx    context.Context
	rexSyncerCancel context.CancelFunc

	//currTask     *SyncTask
	chSyncTask   chan struct{}
	chSyncResult chan *SyncResult

	Status              SyncerStatus
	currContinueFailCnt uint
	currDelay           int

	LastSyncResult *def.RexSyncResult
	lock           sync.Mutex
}

func NewRexLiteSyncer(chainCtx context.Context, grpItem *quorumpb.GroupItem, nodename string, cdnIface def.ChainDataSyncIface, chain *Chain) *RexLiteSyncer {

	rex_syncer_log.Debugf("<%s> NewRexSyncer called", grpItem.GroupId)
	rs := &RexLiteSyncer{}
	rs.GroupItem = grpItem
	rs.GroupId = grpItem.GroupId
	rs.nodename = nodename
	rs.chain = chain
	rs.cdnIface = cdnIface
	rs.chainCtx = chainCtx
	rs.chSyncTask = nil
	rs.chSyncResult = nil
	rs.Status = IDLE
	rs.currContinueFailCnt = 0
	rs.currDelay = 0
	rs.LastSyncResult = nil
	rs.isResultCollecting.Store(false)

	return rs
}

func (rs *RexLiteSyncer) TaskTrigger() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	select {
	case rs.chSyncTask <- struct{}{}:
	case <-ctx.Done():
		rex_syncer_log.Debugf("<%s> task trigger ticker error", rs.GroupId, ctx.Err())
	}
}
func (rs *RexLiteSyncer) AddResult(result *SyncResult) {
	rex_syncer_log.Debugf("<%s> AddResult called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.Status == CLOSED {
		rex_syncer_log.Debugf("<%s> AddResult called, but syncer is closed", rs.GroupId)
		return
	}

	rs.chSyncResult <- result
}

func (rs *RexLiteSyncer) GetLastRexSyncResult() (*def.RexSyncResult, error) {
	rex_syncer_log.Debugf("<%s> GetLastRexSyncResult called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	if rs.LastSyncResult == nil {
		return nil, fmt.Errorf("no valid rex sync result yet")
	}

	return rs.LastSyncResult, nil
}

func (rs *RexLiteSyncer) GetSyncerStatus() SyncerStatus {
	rex_syncer_log.Debugf("<%s> GetSyncerStatus called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	return rs.Status
}

func (rs *RexLiteSyncer) Start() {
	rex_syncer_log.Debugf("<%s> Start called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.rexSyncerCtx != nil {
		rs.rexSyncerCancel()
	}
	rs.Status = RUNNING

	rs.chSyncTask = make(chan struct{})
	rs.chSyncResult = make(chan *SyncResult)
	rs.rexSyncerCtx, rs.rexSyncerCancel = context.WithCancel(rs.chainCtx)
	//start sync worker
	go rs.SyncWorker(rs.chainCtx, rs.chSyncTask)
	taskticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	collectingdone := make(chan struct{})
	//a task listening routine
	taskresults := []*quorumpb.ReqBlockResp{}

	//TEST

	go func() {
		for {
			select {
			case <-rs.chSyncTask:
				//TODO: check timeout?
				var req *quorumpb.ReqBlock
				var reqerr error

				nextBlock := rs.cdnIface.GetCurrBlockId() + uint64(1)
				//trx, trxerr = rs.chain.GetTrxFactory().GetReqBlocksTrx("", rs.GroupId, nextBlock, REQ_BLOCKS_PER_REQUEST)
				userSignKeyname := rs.chain.GetKeynameByPubkey(rs.GroupItem.UserSignPubkey)
				req, reqerr = rumchaindata.GetReqBlocksMsg(rs.GroupId, rs.GroupItem.UserSignPubkey, userSignKeyname, nextBlock, REQ_BLOCKS_PER_REQUEST)

				if reqerr != nil {
					rex_syncer_log.Warningf("<%s> SyncWorker run task get trx failed, err <%s>", rs.GroupId, reqerr.Error())
				}

				blockBundles := &quorumpb.BlocksBundle{}
				block, err := rs.chain.GetBlockFromDSCache(rs.GroupId, nextBlock, rs.nodename)
				if err != nil {
					rex_syncer_log.Debugf("<%s> SyncWorker sync block <%d> from local cache failed, <%s>", rs.GroupId, nextBlock, err.Error())
				} else {
					for block != nil {
						blockBundles.Blocks = append(blockBundles.Blocks, block)
						block, err = rs.chain.GetBlockFromDSCache(rs.GroupId, block.BlockId+1, rs.nodename)
						if err != nil {
							rex_syncer_log.Warningf("<%s> SyncWorker sync from local cache error <%s>", rs.GroupId, err.Error())
						}
					}
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

					rs.currContinueFailCnt = 0
				} else {
					connMgr, err := conn.GetConn().GetConnMgr(rs.GroupId)
					if err != nil {
						rex_syncer_log.Warningf("<%s> SyncWorker run task get connMgr failed, err <%s>", rs.GroupId, err.Error())
					}
					err = connMgr.SendSyncReqMsgRex(req)
					if err != nil {
						rex_syncer_log.Warningf("<%s> SyncWorker run task sendReq failed , err <%s>", rs.GroupId, err.Error())
					}

					rex_syncer_log.Debugf("<%s> ReqTrx send by rex", rs.GroupId)
				}

			case result := <-rs.chSyncResult:
				rex_syncer_log.Debugf("<%s> SyncWorker run task get result", rs.GroupId)
				if rs.isResultCollecting.Load() == false {
					//start a result collecting counter
					go func() {
						randDelay := rand.Intn(500)
						taskDuration := int(rs.currContinueFailCnt)*TASK_DURATION_ADJ + rs.currDelay + randDelay
						if taskDuration > MAXIMUM_TASK_DURATION {
							taskDuration = MAXIMUM_TASK_DURATION
						}
						time.Sleep(time.Duration(taskDuration) * time.Millisecond)
						collectingdone <- struct{}{}
					}()

				}

				//TODO: create a result collector , after the first result received, start collector timer ,wait x secondes
				//ResultCollectorTimer <- taskDuration
				//NO need to check block, block will be valid later
				//check if the resp is what we are waiting for

				reqBlockResp := result.Data.(*quorumpb.ReqBlockResp)
				rex_syncer_log.Debugf("- Receive valid reqBlockResp, provider <%s> result <%s> from block <%d> total <%d> blocks provided",
					reqBlockResp.ProviderPubkey,
					reqBlockResp.Result.String(),
					reqBlockResp.FromBlock,
					len(reqBlockResp.Blocks.Blocks))
				//add valid result to list
				taskresults = append(taskresults, reqBlockResp)
			case <-collectingdone:
				rex_syncer_log.Debugf("<%s> SyncWorker run task done", rs.GroupId)

				//no result found, timeout
				if len(taskresults) == 0 {
					rex_syncer_log.Debugf("<%s> SyncWorker run task timeout, no result", rs.GroupId)
					rs.currContinueFailCnt += 1
				} else {
					//select a "winner" response
					//1. choose resp provided the most blocks
					//2. if same, choose response from producers
					rex_syncer_log.Debugf("<%s> SyncWorker run task select winner", rs.GroupId)
					var winnerResp *quorumpb.ReqBlockResp
					for _, resp := range taskresults {
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

					//TODO: remove items instead of create new array
					taskresults = []*quorumpb.ReqBlockResp{}

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
					//set ResultCollecting status to false, task done.
					rs.isResultCollecting.Store(false)
				}
				//}
				//TODO: quit
				//case <-quit:
				//	return
			}

		}
	}()

	// a task trigger routine
	go func() {
		for {
			select {
			case <-taskticker.C:
				rs.TaskTrigger()
				td := rs.GetCurrentTaskDurationAdj()
				if td > 0 {
					taskticker.Reset(time.Duration(rs.GetCurrentTaskDurationAdj()) * time.Second)
				}
			case <-quit:
				return
			}
		}
	}()

	rs.TriggerSyncTask()
}

func (rs *RexLiteSyncer) TriggerSyncTask() {
	// 500 ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*5000))
	defer cancel()
	select {
	case rs.chSyncTask <- struct{}{}:
		errorMsg := string("")
		if ctx.Err() != nil {
			errorMsg = ctx.Err().Error()
		}

		rex_syncer_log.Debugf("<%s> fire a task, err <%s>", rs.GroupId, errorMsg)
	case <-ctx.Done():
		errorMsg := string("")
		if ctx.Err() != nil {
			errorMsg = ctx.Err().Error()
		}
		rex_syncer_log.Debugf("<%s> task trigger ticker timeout: err <%s>", rs.GroupId, errorMsg)
	}
}

func (rs *RexLiteSyncer) Stop() {
	rex_syncer_log.Debugf("<%s> Stop called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.Status = CLOSED

	if rs.rexSyncerCancel != nil {
		rs.rexSyncerCancel()
	}

	//close(rs.chSyncTask)
}

func (rs *RexLiteSyncer) SyncWorker(chainCtx context.Context, chSyncTask <-chan struct{}) {
	rex_syncer_log.Debugf("<%s> SyncWorker called", rs.GroupId)
	for {
		select {
		case <-chainCtx.Done():
			rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
			return
		case <-rs.rexSyncerCtx.Done():
			rex_syncer_log.Debugf("<%s> SyncWorker exit", rs.GroupId)
			return
		}
	}
}

func (rs *RexLiteSyncer) GetCurrentTaskDurationAdj() int {
	rex_syncer_log.Debugf("<%s> GetCurrentDelay called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()
	currAdj := int(rs.currContinueFailCnt)*TASK_DURATION_ADJ + rs.currDelay

	if currAdj > MAXIMUM_TASK_DURATION {
		currAdj = MAXIMUM_TASK_DURATION
	}

	return currAdj
}

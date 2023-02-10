package chain

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"

	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
)

var TASK_RETRY_NUM = 30                // task retry times
var REQ_BLOCKS_PER_REQUEST = 10        // ask for n blocks per request
var SYNC_BLOCK_TASK_TIMEOUT = 4 * 1000 // in millseconds
var SYNC_BLOCK_FREQ_ADJ = 5 * 1000     // in millseconds
var MAXIMUM_DELAY_DURATION = 60 * 1000 // is millseconds

var rex_syncer_log = logging.Logger("rsyncer")

type SyncResult struct {
	TaskId int64
	Data   interface{}
}

type SyncerStatus uint

const (
	IDLE SyncerStatus = iota
	SYNCING
	CLOSED
)

type SyncTask struct {
	TaskId      int64 //epoch
	ReqBlockNum int
	DelayTime   int
	TriggerTime int64
}

type RexSyncer struct {
	GroupId  string
	nodename string
	cdnIface def.ChainDataSyncIface
	chainCtx *Chain

	//chan signals
	taskq      chan *SyncTask
	resultq    chan *SyncResult
	taskdone   chan struct{}
	stopnotify chan struct{}

	Status         SyncerStatus
	CurrRetryCount uint
	CurrentDely    int
	CurrentTask    *SyncTask

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
	rs.taskdone = make(chan struct{})
	rs.stopnotify = make(chan struct{})

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
			//calculate current delay
			task.DelayTime += int(rs.CurrRetryCount)*SYNC_BLOCK_FREQ_ADJ + rs.CurrentDely
			if task.DelayTime > MAXIMUM_DELAY_DURATION {
				task.DelayTime = MAXIMUM_DELAY_DURATION
			}

			task.TriggerTime = time.Now().Unix() + int64(task.DelayTime)/1000
			taskTimeout := task.DelayTime + SYNC_BLOCK_TASK_TIMEOUT
			rex_syncer_log.Debugf("<%s> get task <%d> from taskq, set task timeout to <%d>", rs.GroupId, task.TaskId, taskTimeout)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(taskTimeout)*time.Millisecond)
			defer cancel()
			rs.runTask(ctx, task)
		}
		rs.stopnotify <- struct{}{}
	}()

	//start resultq
	go func() {
		for result := range rs.resultq {
			rs.handleResult(result)
		}
		rs.stopnotify <- struct{}{}
	}()

	task := rs.newSyncBlockTask()
	rs.AddTask(task)
}

func (rs *RexSyncer) Stop() {
	rex_syncer_log.Debugf("<%s> Stop called", rs.GroupId)
	rs.Status = CLOSED
	safeCloseTaskQ(rs.taskq)
	safeCloseResultQ(rs.resultq)
	safeClose(rs.taskdone)
	if rs.stopnotify != nil {
		signcount := 0
		for range rs.stopnotify {
			signcount++
			//wait stop sign and set idle
			if signcount == 2 { // taskq and resultq stopped
				close(rs.stopnotify)
				rex_syncer_log.Debugf("<%s> rexsyncer stop success.", rs.GroupId)
			}
		}
	}
}
func (rs *RexSyncer) GetLastRexSyncResult() (*def.RexSyncResult, error) {
	if rs.LastSyncResult == nil {
		return nil, fmt.Errorf("no valid rex sync result yet")
	}

	if rs.CurrentTask != nil {
		rs.LastSyncResult.NextSyncTaskTimeStamp = int(rs.CurrentTask.TriggerTime)
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

func (rs *RexSyncer) runTask(ctx context.Context, task *SyncTask) error {
	//TODO: close this goroutine when the processTask func return. add some defer signal?
	rex_syncer_log.Debugf("runTask called, taskId <%d>, retry <%d>", task.TaskId, rs.CurrRetryCount)
	go func() {
		rs.CurrentTask = task //set current task
		rs.syncBlockTaskSender(task)
	}()

	select {
	case <-rs.taskdone:
		return nil
	case <-ctx.Done():
		if rs.Status != CLOSED {
			//a workround, should cancel the ctx for current task
			if rs.CurrentTask != nil {
				rex_syncer_log.Debugf("task <%d> timeout", task.TaskId)
				rs.CurrRetryCount += 1
				rex_syncer_log.Debugf("CurrRetryCount <%d>", rs.CurrRetryCount)

				//close current task
				rs.taskdone <- struct{}{}
				rs.CurrentTask = nil
				rs.Status = IDLE

				//start next round with highest epoch
				task := rs.newSyncBlockTask()
				rs.AddTask(task)
			}
		}
		return nil
	}
}

func (rs *RexSyncer) AddTask(task *SyncTask) {
	rex_syncer_log.Debugf("Gsyncer addTask called")
	go func() {
		if rs.Status != CLOSED {
			rs.taskq <- task
		}
	}()
}

func (rs *RexSyncer) AddResult(result *SyncResult) {
	go func() {
		if rs.Status != CLOSED {
			rs.resultq <- result
		}
	}()
}

// task generators
func (rs *RexSyncer) newSyncBlockTask() *SyncTask {
	rex_syncer_log.Debugf("<%s> newSyncBlockTask called", rs.GroupId)
	nextEpoch := rs.cdnIface.GetCurrEpoch() + 1
	randDelay := rand.Intn(500)
	return &SyncTask{TaskId: nextEpoch, ReqBlockNum: REQ_BLOCKS_PER_REQUEST, DelayTime: randDelay}
}

func (rs *RexSyncer) syncBlockTaskSender(task *SyncTask) error {
	rex_syncer_log.Debugf("<%s> syncBlockTaskSender called", rs.GroupId)

	var trx *quorumpb.Trx
	var trxerr error

	trx, trxerr = rs.chainCtx.GetTrxFactory().GetReqBlocksTrx("", rs.GroupId, task.TaskId, int64(task.ReqBlockNum))
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

	rex_syncer_log.Debugf("- Receive valid reqBlockResp, provider <%s> result <%s> from epoch <%d> total blocks provided <%d>",
		reqBlockResp.ProviderPubkey,
		reqBlockResp.Result.String(),
		reqBlockResp.FromEpoch,
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
		FromEpoch:             reqBlockResp.FromEpoch,
		BlockProvided:         reqBlockResp.BlksProvided,
		SyncResult:            reqBlockResp.Result.String(),
		LastSyncTaskTimestamp: time.Now().Unix(),
		NextSyncTaskTimeStamp: -1,
	}

	rs.taskdone <- struct{}{}
	rs.CurrentTask = nil
	rs.Status = IDLE

	//start next round
	task := rs.newSyncBlockTask()
	rs.AddTask(task)

	return nil
}

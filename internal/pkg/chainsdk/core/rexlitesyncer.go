package chain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
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
	GroupItem *quorumpb.GroupItem
	GroupId   string
	nodename  string
	chain     *Chain
	cdnIface  def.ChainDataSyncIface
	chainCtx  context.Context

	rexSyncerCtx    context.Context
	rexSyncerCancel context.CancelFunc

	currTask *SyncTask
	//chSyncTask chan *SyncTask
	chSyncTask chan struct{}

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
	rs.currTask = nil
	rs.chSyncTask = nil
	rs.Status = IDLE
	rs.currContinueFailCnt = 0
	rs.currDelay = 0
	rs.LastSyncResult = nil

	return rs
}

func (rs *RexLiteSyncer) AddResult(result *SyncResult) {
	rex_syncer_log.Debugf("<%s> AddResult called", rs.GroupId)
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.Status == CLOSED {
		rex_syncer_log.Debugf("<%s> AddResult called, but syncer is closed", rs.GroupId)
		return
	}
	rex_syncer_log.Debugf("<%s> AddResult Not implemented in LiteSyncer", rs.GroupId)

	//if rs.currTask != nil {
	//	rs.currTask.chSyncResult <- result
	//}
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

	rex_syncer_log.Debugf("<%s> Not implemented start", rs.GroupId)
	rs.chSyncTask = make(chan struct{})
	rs.rexSyncerCtx, rs.rexSyncerCancel = context.WithCancel(rs.chainCtx)

	//start sync worker
	go rs.SyncWorker(rs.chainCtx, rs.chSyncTask)
	taskticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-taskticker.C:
				rs.chSyncTask <- struct{}{}
				td := rs.GetCurrentTaskDurationAdj()
				if td > 0 {
					taskticker.Reset(time.Duration(rs.GetCurrentTaskDurationAdj()) * time.Second)
				}
			case <-quit:
				return
			}
		}
	}()
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
			//if rs.currTask != nil {
			//	rex_syncer_log.Debugf("<%s> SyncWorker cancel current task", rs.GroupId)
			//	rs.currTask.TaskCancel()
			//	rs.currTask = nil
			//}
			return

		case <-chSyncTask:
			fmt.Println("sync run")
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

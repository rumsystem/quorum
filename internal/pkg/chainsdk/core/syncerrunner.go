package chain

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var WAIT_BLOCK_TIME_S = 10 //wait time period
var RETRY_LIMIT = 30       //retry times

const (
	SYNCING_FORWARD  = 0
	SYNCING_BACKWARD = 1
	SYNC_FAILED      = 2
	IDLE             = 3
	LOCAL_SYNCING    = 4
)

type SyncerRunner struct {
	nodeName string
	group    *Group
	Status   int8
	//statusBeforeFail    int8
	//responses           map[string]*quorumpb.ReqBlockResp
	//blockReceived       map[string]string
	//direction       Syncdirection
	currenttaskid   string
	currentWaitTask *EpochSyncTask
	taskserialid    uint32
	resultserialid  uint32
	cdnIface        def.ChainDataSyncIface
	syncNetworkType conn.P2pNetworkType
	gsyncer         *Gsyncer
	//rwMutex         sync.RWMutex
	//localSyncFinished   bool
	rumExchangeTestMode bool
}

func NewSyncerRunner(group *Group, cdnIface def.ChainDataSyncIface, nodename string) *SyncerRunner {
	gsyncer_log.Debugf("<%s> NewSyncerRunner called", group.Item.GroupId)
	sr := &SyncerRunner{}
	sr.group = group
	sr.cdnIface = cdnIface
	sr.taskserialid = 0
	sr.resultserialid = 0
	//sr.direction = Next
	sr.Status = IDLE
	sr.cdnIface = cdnIface
	sr.syncNetworkType = conn.PubSub
	sr.rumExchangeTestMode = false

	gs := NewGsyncer(group.Item.GroupId, sr.GetEpochTask, sr.ResultReceiver, sr.TaskSender)
	sr.gsyncer = gs
	gsyncer_log.Debugf("<%s> NewSyncerRunner initialed", group.Item.GroupId)
	return sr

}

func (sr *SyncerRunner) SetRumExchangeTestMode() {
	sr.rumExchangeTestMode = true
}

func (sr *SyncerRunner) GetWaitEpoch() int64 {
	return sr.gsyncer.GetWaitEpoch()
}

//define how to get next task, for example, taskid+1
func (sr *SyncerRunner) GetEpochTask(epoch int64) (*EpochSyncTask, error) {
	if epoch == 0 {
		return nil, errors.New("No task for Epoch 0 ")
	} else {
		return &EpochSyncTask{Epoch: epoch}, nil
	}
	return nil, nil
	//if blockid == "" { //workaround, return current task id to retry
	//	blockid = sr.currenttaskid
	//} else if blockid == "0" { // warkaround for Rex sync, forward only
	//	taskmeta := BlockSyncTask{BlockId: sr.group.Item.HighestBlockId, Direction: Next}
	//	taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
	//	return &SyncTask{Meta: taskmeta, Id: taskid}, nil
	//} else {
	//	sr.currenttaskid = blockid
	//}

	//sr.taskserialid++
	//taskmeta := BlockSyncTask{BlockId: blockid, Direction: sr.direction}
	//taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
	//return &SyncTask{Meta: taskmeta, Id: taskid}, nil
}

//func (sr *SyncerRunner) StartBackward(blockid string) error {
//	//backward sync
//	sr.Status = SYNCING_BACKWARD
//	sr.direction = Previous
//	task, err := sr.GetBlockTask(blockid)
//	if err != nil {
//		return err
//	}
//	sr.gsyncer.Start()
//	//add the first task
//	sr.gsyncer.AddTask(task)
//	return nil
//}

func (sr *SyncerRunner) Start(epoch int64) error {
	//default forward sync
	sr.Status = SYNCING_FORWARD
	//sr.direction = Next
	task, err := sr.GetEpochTask(epoch)
	if err != nil {
		return err
	}
	sr.gsyncer.Start()
	//add the first task
	sr.gsyncer.AddTask(task)
	return nil
}

//func (sr *SyncerRunner) SwapSyncDirection() {
//	if sr.Status == SYNCING_FORWARD {
//		sr.Status = SYNCING_BACKWARD
//		sr.direction = Previous
//	} else if sr.Status == SYNCING_BACKWARD {
//		sr.Status = SYNCING_FORWARD
//		sr.direction = Next
//	}
//}

func (sr *SyncerRunner) Stop() {
	sr.Status = IDLE
	sr.gsyncer.Stop()
}

//func (sr *SyncerRunner) SetCurrentWaitTask(task *BlockSyncTask) {
//	sr.currentWaitTask = task
//}

func (sr *SyncerRunner) TaskSender(task *EpochSyncTask) error {
	gsyncer_log.Debugf("<%s> call TaskSender... with epoch: %d", sr.group.Item.GroupId, task.Epoch)

	var trx *quorumpb.Trx
	var trxerr error

	trx, trxerr = sr.group.ChainCtx.GetTrxFactory().GetReqBlockForwardTrxWithEpoch("", task.Epoch, sr.group.Item.GroupId)
	if trxerr != nil {
		return trxerr
	}

	connMgr, err := conn.GetConn().GetConnMgr(sr.group.Item.GroupId)
	if err != nil {
		return err
	}
	gsyncer_log.Debugf("<%s> ======TODO: set current wait task", sr.group.Item.GroupId)
	//sr.SetCurrentWaitTask(&blocktask)

	gsyncer_log.Debugf("<%s> ======TODO: check RetryCounter", sr.group.Item.GroupId)

	//	if sr.gsyncer.RetryCounter() >= 30 { //max retry count
	//		//change networktype and clear counter
	//		if sr.rumExchangeTestMode != true {

	//			if sr.syncNetworkType == conn.PubSub {
	//				sr.syncNetworkType = conn.RumExchange
	//			} else {
	//				sr.syncNetworkType = conn.PubSub
	//			}
	//			gsyncer_log.Debugf("<%s> retry counter %d, change the network type to %s", sr.gsyncer.RetryCounter(), sr.syncNetworkType)
	//		}
	//		sr.gsyncer.RetryCounterClear()
	//	}
	v := rand.Intn(500)
	time.Sleep(time.Duration(v) * time.Millisecond) // add some random delay
	gsyncer_log.Debugf("<%s> ======TODO: send trx by pubsub or rex", sr.group.Item.GroupId)
	//if sr.rumExchangeTestMode == false && sr.syncNetworkType == conn.PubSub {
	return connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
	//} else {
	//	return connMgr.SendTrxRex(trx, nil)
	//}
	return nil
}
func (sr *SyncerRunner) ResultReceiver(result *SyncResult) (int64, error) {
	trxtaskresult, ok := result.Data.(*quorumpb.Trx)
	if ok == true {
		//v := rand.Intn(5) + 1
		//time.Sleep(time.Duration(v) * time.Second) // fake workload
		//try to save the result to db
		nextepoch, err := sr.group.ChainCtx.HandleReqBlockResp(trxtaskresult)
		if err != nil {
			if err == ErrSyncDone {
				sr.Status = IDLE
			} else if err.Error() == "PARENT_NOT_EXIST" && sr.Status == SYNCING_BACKWARD {
				gsyncer_log.Debugf("<%s> PARENT_NOT_EXIST, continue. %s", sr.group.Item.GroupId, result.Id)
				//err = nil
			} else {
				err = ErrNotAccept
			}
		}

		//	//workaround change the return of rumexchage result to ErrIgnore
		//	if sr.syncNetworkType == conn.RumExchange || sr.rumExchangeTestMode == true {
		//		return "", ErrIgnore
		//	}
		return nextepoch, err
	} else {
		gsyncer_log.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
		return 0, fmt.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
	}
}

func (sr *SyncerRunner) AddTrxToSyncerQueue(trx *quorumpb.Trx) {
	sr.resultserialid++
	resultid := strconv.FormatUint(uint64(sr.resultserialid), 10)
	result := &SyncResult{Id: resultid, Data: trx}
	sr.gsyncer.AddResult(result)
}

package chain

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var WAIT_BLOCK_TIME_S = 10 //wait time period
var RETRY_LIMIT = 30       //retry times

const (
	SYNCING_FORWARD  = 0
	SYNCING_BACKWARD = 1
	SYNC_FAILED      = 2
	IDLE             = 3
	LOCAL_SYNCING    = 4
	CLOSE            = 5
)

type SyncerRunner struct {
	nodeName string
	group    *Group
	Status   int8
	//responses           map[string]*quorumpb.ReqBlockResp
	currenttaskid   string
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
	sr.Status = IDLE
	sr.cdnIface = cdnIface
	sr.syncNetworkType = conn.PubSub
	sr.rumExchangeTestMode = false

	gs := NewGsyncer(group.Item.GroupId, sr.GetEpochTask, sr.ResultReceiver, sr.TaskSender)
	gs.SetRetryWithNext(false)
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
func (sr *SyncerRunner) GetEpochTask(epoch int64) (*SyncTask, error) {
	//TODO
	//if blockid == "" { //workaround, return current task id to retry
	//	blockid = sr.currenttaskid
	//} else
	//if blockid == "0" { // warkaround for Rex sync, forward only
	//	taskmeta := BlockSyncTask{BlockId: sr.group.Item.HighestBlockId, Direction: Next}
	//	taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
	//	return &SyncTask{Meta: taskmeta, Id: taskid}, nil
	//} else {
	//	sr.currenttaskid = blockid
	//}
	if epoch == 0 {
		return nil, errors.New("No task for Epoch 0 ")
	} else {
		taskmeta := EpochSyncTask{Epoch: epoch}
		taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
		return &SyncTask{Meta: taskmeta, Id: taskid}, nil
	}
}

func (sr *SyncerRunner) Start(epoch int64) error {
	//default forward sync
	sr.Status = SYNCING_FORWARD
	task, err := sr.GetEpochTask(epoch)
	if err != nil {
		return err
	}
	sr.gsyncer.Start()
	//add the first task
	sr.gsyncer.addTask(task)
	return nil
}

func (sr *SyncerRunner) Stop() {
	sr.Status = IDLE
	sr.gsyncer.Stop()
}

func (sr *SyncerRunner) TaskSender(task *SyncTask) error {
	//TODO
	//if sr.syncNetworkType == conn.RumExchange || sr.rumExchangeTestMode == true {
	//	sr.gsyncer.SetRetryWithNext(true) //workaround for rumexchange
	//}
	blocktask, ok := task.Meta.(EpochSyncTask)

	if ok == true {
		gsyncer_log.Debugf("<%s> call TaskSender... with BlockId: %s", sr.group.Item.GroupId, blocktask.Epoch)
		//TODO: keep a block task lock

		//block, err := sr.group.GetBlock(blocktask.Epoch)
		//var trx *quorumpb.Trx
		//var trxerr error

		//if blocktask.Direction == Next {
		//	trx, trxerr = sr.group.ChainCtx.GetTrxFactory().GetReqBlockForwardTrx("", block)
		//}

		//if trxerr != nil {
		//	return trxerr
		//}

		var trx *quorumpb.Trx
		var trxerr error

		trx, trxerr = sr.group.ChainCtx.GetTrxFactory().GetReqBlockForwardTrxWithEpoch("", blocktask.Epoch, sr.group.Item.GroupId)
		if trxerr != nil {
			return trxerr
		}

		connMgr, err := conn.GetConn().GetConnMgr(sr.group.Item.GroupId)
		if err != nil {
			return err
		}
		//TODO
		//sr.SetCurrentWaitTask(&blocktask)

		if sr.gsyncer.RetryCounter() >= 30 { //max retry count
			//change networktype and clear counter
			if sr.rumExchangeTestMode != true {

				if sr.syncNetworkType == conn.PubSub {
					sr.syncNetworkType = conn.RumExchange
				} else {
					sr.syncNetworkType = conn.PubSub
				}
				gsyncer_log.Debugf("<%s> retry counter %d, change the network type to %s", sr.gsyncer.RetryCounter(), sr.syncNetworkType)
			}
			sr.gsyncer.RetryCounterClear()
		}

		v := rand.Intn(500)
		time.Sleep(time.Duration(v) * time.Millisecond) // add some random delay
		if sr.rumExchangeTestMode == false && sr.syncNetworkType == conn.PubSub {
			return connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
		} else {
			//send the request, will wait for the response
			return connMgr.SendReqTrxRex(trx)
		}
	} else {
		gsyncer_log.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
		return fmt.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
	}
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
				gsyncer_log.Debugf("<%s> PARENT_NOT_EXIST %s", sr.group.Item.GroupId, result.Id)
				//err = nil
			}
		}
		return nextepoch, err
	} else {
		gsyncer_log.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
		return 0, fmt.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
	}
}

func (sr *SyncerRunner) AddTrxToSyncerQueue(trx *quorumpb.Trx, peerid peer.ID) {
	sr.resultserialid++
	resultid := strconv.FormatUint(uint64(sr.resultserialid), 10)
	result := &SyncResult{Id: resultid, Data: trx}
	if sr.Status == SYNCING_FORWARD || sr.Status == SYNCING_BACKWARD {
		sr.gsyncer.AddResult(result)
	}
}

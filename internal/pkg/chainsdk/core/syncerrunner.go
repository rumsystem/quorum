package chain

import (
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
	//AskNextTimer        *time.Timer
	//AskNextTimerDone    chan bool
	Status int8
	//retryCount int8
	//statusBeforeFail    int8
	//responses           map[string]*quorumpb.ReqBlockResp
	//blockReceived       map[string]string
	direction       Syncdirection
	currenttaskid   string
	taskserialid    uint32
	resultserialid  uint32
	cdnIface        def.ChainDataSyncIface
	syncNetworkType conn.P2pNetworkType
	gsyncer         *Gsyncer
	//rwMutex         sync.RWMutex
	//localSyncFinished   bool
	//rumExchangeTestMode bool
}

func NewSyncerRunner(group *Group, cdnIface def.ChainDataSyncIface, nodename string) *SyncerRunner {
	gsyncer_log.Debugf("<%s> NewSyncerRunner called", group.Item.GroupId)
	sr := &SyncerRunner{}
	sr.group = group
	sr.cdnIface = cdnIface
	sr.taskserialid = 0
	sr.resultserialid = 0
	sr.direction = Next
	sr.Status = IDLE
	sr.cdnIface = cdnIface
	sr.syncNetworkType = conn.PubSub
	gs := NewGsyncer(group.Item.GroupId, sr.GetBlockTask, sr.ResultReceiver, sr.TaskSender)
	sr.gsyncer = gs
	gsyncer_log.Debugf("<%s> NewSyncerRunner initialed", group.Item.GroupId)
	return sr

}

//define how to get next task, for example, taskid+1
func (sr *SyncerRunner) GetBlockTask(blockid string) (*SyncTask, error) {
	if blockid == "" { //workaround, return current task id to retry
		blockid = sr.currenttaskid
	} else {
		sr.currenttaskid = blockid
	}
	sr.taskserialid++
	taskmeta := BlockSyncTask{BlockId: blockid, Direction: sr.direction}
	taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
	return &SyncTask{Meta: taskmeta, Id: taskid}, nil
}

func (sr *SyncerRunner) StartBackward(blockid string) error {
	//backward sync
	sr.Status = SYNCING_BACKWARD
	sr.direction = Previous
	task, err := sr.GetBlockTask(blockid)
	if err != nil {
		return err
	}
	sr.gsyncer.Start()
	//add the first task
	sr.gsyncer.AddTask(task)
	return nil
}

func (sr *SyncerRunner) Start(blockid string) error {
	//default forward sync
	sr.Status = SYNCING_FORWARD
	sr.direction = Next
	task, err := sr.GetBlockTask(blockid)
	if err != nil {
		return err
	}
	sr.gsyncer.Start()
	//add the first task
	sr.gsyncer.AddTask(task)
	return nil
}

func (sr *SyncerRunner) Stop() {
	sr.gsyncer.Stop()
	sr.Status = IDLE
}

func (sr *SyncerRunner) TaskSender(task *SyncTask) error {
	gsyncer_log.Debugf("<%s> call TaskSender...", sr.group.Item.GroupId)
	blocktask, ok := task.Meta.(BlockSyncTask)
	if ok == true {
		v := rand.Intn(500)
		time.Sleep(time.Duration(v) * time.Millisecond) // add some random delay

		block, err := sr.group.GetBlock(blocktask.BlockId)
		if sr.Status == SYNCING_BACKWARD && block == nil {
			gsyncer_log.Debugf("<%s> backward sync, can't get block form db, make new block.", sr.group.Item.GroupId)
			err = nil
			block = &quorumpb.Block{GroupId: sr.group.Item.GroupId, BlockId: blocktask.BlockId}
		}
		if err != nil {
			return err
		}

		var trx *quorumpb.Trx
		var trxerr error

		if blocktask.Direction == Next {
			trx, trxerr = sr.group.ChainCtx.GetTrxFactory().GetReqBlockForwardTrx("", block)
		} else {
			trx, trxerr = sr.group.ChainCtx.GetTrxFactory().GetReqBlockBackwardTrx("", block)
		}

		if trxerr != nil {
			return trxerr
		}

		connMgr, err := conn.GetConn().GetConnMgr(sr.group.Item.GroupId)
		if err != nil {
			return err
		}
		//TOFIX: rumExchange
		//if syncer.rumExchangeTestMode == true {
		//	return connMgr.SendTrxRex(trx, nil)
		//}

		//if syncer.syncNetworkType == conn.PubSub {
		return connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
		//} else {
		//	return connMgr.SendTrxRex(trx, nil)
		//}
	} else {
		gsyncer_log.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
		return fmt.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
	}
	return nil
}
func (sr *SyncerRunner) ResultReceiver(result *SyncResult) (string, error) {
	trxtaskresult, ok := result.Data.(*quorumpb.Trx)
	if ok == true {
		//v := rand.Intn(5) + 1
		//time.Sleep(time.Duration(v) * time.Second) // fake workload
		//try to save the result to db
		nexttaskid, err := sr.group.ChainCtx.HandleReqBlockResp(trxtaskresult)
		if err != nil {
			if err == ErrSyncDone {
				sr.Status = IDLE
			}
			if err.Error() == "PARENT_NOT_EXIST" && sr.Status == SYNCING_BACKWARD {
				gsyncer_log.Debugf("<%s> PARENT_NOT_EXIST and SYNCING_BACKWARD, continue. %s", sr.group.Item.GroupId, result.Id)
				err = nil
			}
		} else {
			fmt.Printf("===============no error, nextblockid %s genesisblockid %s \n", nexttaskid, sr.group.Item.GenesisBlock.BlockId)
			if sr.Status == SYNCING_BACKWARD && nexttaskid == sr.group.Item.GenesisBlock.BlockId {
				fmt.Printf("===============ok to stop sync ,set forward add a forward task to try")
			}
		}

		return nexttaskid, err
	} else {
		gsyncer_log.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
		return "", fmt.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
	}
}

func (sr *SyncerRunner) AddTrxToSyncerQueue(trx *quorumpb.Trx) {
	sr.resultserialid++
	resultid := strconv.FormatUint(uint64(sr.resultserialid), 10)
	result := &SyncResult{Id: resultid, Data: trx}
	sr.gsyncer.AddResult(result)
}

package chain

import (
	"fmt"
	"strconv"

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

func (sr *SyncerRunner) Start(blockid string) error {
	fmt.Println("=========start...syncer with...", blockid)
	//default forward sync
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
}

func (sr *SyncerRunner) TaskSender(task *SyncTask) error {
	gsyncer_log.Debugf("<%s> call TaskSender...", sr.group.Item.GroupId)
	blocktask, ok := task.Meta.(BlockSyncTask)
	if ok == true {
		block, err := sr.group.GetBlock(blocktask.BlockId)
		if err != nil {
			return err
		}
		trx, err := sr.group.ChainCtx.GetTrxFactory().GetReqBlockForwardTrx("", block)
		if err != nil {
			return err
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
		return sr.group.ChainCtx.HandleReqBlockResp(trxtaskresult)
	} else {
		gsyncer_log.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
		return "", fmt.Errorf("<%s> Unsupported result %s", sr.group.Item.GroupId, result.Id)
	}
}

func (sr *SyncerRunner) AddTrxToSyncerQueue(trx *quorumpb.Trx) {
	//type SyncResult struct {
	//Data   interface{}
	//Id     string
	//TaskId string
	//data := BlockSyncResult{BlockId: fmt.Sprintf("test_block_id_%s", task.Id)}
	sr.resultserialid++
	resultid := strconv.FormatUint(uint64(sr.resultserialid), 10)
	result := &SyncResult{Id: resultid, Data: trx}
	sr.gsyncer.AddResult(result)
}

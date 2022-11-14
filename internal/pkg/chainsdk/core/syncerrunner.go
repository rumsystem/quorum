package chain

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var syncerrunner_log = logging.Logger("syncerrunner")

var WAIT_BLOCK_TIME_S = 10 //wait time period
var RETRY_LIMIT = 30       //retry times

const (
	SYNCING_FORWARD = 0
	SYNC_FAILED     = 1
	IDLE            = 2
	CLOSE           = 3
	LOCAL_SYNCING   = 4
	CONSENSUS_SYNC  = 5
)

type SyncerRunner struct {
	group  *Group
	Status int8

	currenttaskid  string
	taskserialid   uint32
	resultserialid uint32

	cdnIface        def.ChainDataSyncIface
	syncNetworkType conn.P2pNetworkType
	gsyncer         *Gsyncer

	rumExchangeTestMode bool
	//nodeName string
	//responses           map[string]*quorumpb.ReqBlockResp
	//rwMutex         sync.RWMutex
	//localSyncFinished   bool

}

func NewSyncerRunner(group *Group, cdnIface def.ChainDataSyncIface, nodename string) *SyncerRunner {
	syncerrunner_log.Debugf("<%s> NewSyncerRunner called", group.Item.GroupId)
	sr := &SyncerRunner{}
	sr.group = group
	sr.cdnIface = cdnIface
	sr.taskserialid = 0
	sr.resultserialid = 0
	sr.Status = IDLE
	sr.cdnIface = cdnIface
	sr.syncNetworkType = conn.PubSub
	sr.rumExchangeTestMode = false

	//create and initial Get Task Apis
	APIMap := make(map[TaskType]func(args ...interface{}) (*SyncTask, error))
	APIMap[GetEpoch] = sr.GetEpochTask
	APIMap[ConsensusSync] = sr.GetConsensusSyncTask

	gs := NewGsyncer(group.Item.GroupId, APIMap, sr.ResultReceiver, sr.TaskSender)
	gs.SetRetryWithNext(false)
	sr.gsyncer = gs

	return sr
}

func (sr *SyncerRunner) SetRumExchangeTestMode() {
	syncerrunner_log.Debugf("<%s> SetRumExchangeTestMode called", sr.group.Item.GroupId)
	sr.rumExchangeTestMode = true
}

func (sr *SyncerRunner) GetWaitEpoch() int64 {
	syncerrunner_log.Debugf("<%s> GetWaitEpoch called", sr.group.Item.GroupId)
	return sr.gsyncer.GetWaitEpoch()
}

// define how to get next task, for example, taskid+1
func (sr *SyncerRunner) GetEpochTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetEpochTask called", sr.group.Item.GroupId)
	//convert func parameter

	if len(args) != 1 {
		return nil, fmt.Errorf("params mismatch, GetEpochTask(epoch int64)")
	}
	epoch, ok := args[0].(int64)
	if !ok {
		return nil, fmt.Errorf("convert interface{} to int64 failed")
	}

	if epoch == 0 {
		return nil, errors.New("no task for Epoch 0 ")
	} else {
		taskmeta := EpochSyncTask{Epoch: epoch}
		taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
		return &SyncTask{Type: GetEpoch, Meta: taskmeta, Id: taskid}, nil
	}
}

func (sr *SyncerRunner) GetConsensusSyncTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetConsensusSyncTask called", sr.group.Item.GroupId)
	taskmate := ConsensusSyncTask{SessionId: uuid.NewString()}
	taskid := strconv.FormatUint(uint64(sr.taskserialid), 10)
	return &SyncTask{Type: ConsensusSync, Meta: taskmate, Id: taskid}, nil
}

func (sr *SyncerRunner) Start() error {
	syncerrunner_log.Debugf("<%s> Start called", sr.group.Item.GroupId)

	var task *SyncTask
	var err error
	//Check if producer node
	if _, ok := sr.group.ChainCtx.ProducerPool[sr.group.Item.UserSignPubkey]; ok {
		//producer try get consensus before start sync block
		groupMgr_log.Debugf("<%s> producer(owner) node try get consensus before sync", sr.group.Item.GroupId)
		sr.Status = CONSENSUS_SYNC
		task, err = sr.GetConsensusSyncTask()
		if err != nil {
			return err
		}
	} else {
		groupMgr_log.Debugf("<%s> user node start epoch (block) sync. group", sr.group.Item.GroupId)
		sr.Status = SYNCING_FORWARD
		task, err = sr.GetEpochTask(sr.group.Item.Epoch + 1)
		if err != nil {
			return err
		}
	}

	//start syncer
	sr.gsyncer.Start()

	//add the first task
	sr.gsyncer.addTask(task)
	return nil
}

func (sr *SyncerRunner) Stop() {
	syncerrunner_log.Debugf("<%s> Stop called", sr.group.Item.GroupId)
	sr.Status = IDLE
	sr.gsyncer.Stop()
}

func (sr *SyncerRunner) TaskSender(task *SyncTask) error {
	syncerrunner_log.Debugf("<%s> TaskSender called", sr.group.Item.GroupId)

	//TODO
	//if sr.syncNetworkType == conn.RumExchange || sr.rumExchangeTestMode == true {
	//	sr.gsyncer.SetRetryWithNext(true) //workaround for rumexchange
	//}

	if task.Type == GetEpoch {
		blocktask, ok := task.Meta.(EpochSyncTask)
		if !ok {
			gsyncer_log.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
			return fmt.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
		}
		syncerrunner_log.Debugf("<%s> TaskSender with GetEpoch Task, Epoch <%d>", sr.group.Item.GroupId, blocktask.Epoch)
		//TODO: keep a block task lock

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
			if !sr.rumExchangeTestMode {
				if sr.syncNetworkType == conn.PubSub {
					sr.syncNetworkType = conn.RumExchange
				} else {
					sr.syncNetworkType = conn.PubSub
				}
				syncerrunner_log.Debugf("<%s> retry <%d> timers, switch network type to <%s>", sr.group.Item.GroupId, sr.gsyncer.RetryCounter(), sr.syncNetworkType)
			}
			sr.gsyncer.RetryCounterClear()
		}

		//Commented by cuicat
		//?? Do we need this in "real" network environment??
		v := rand.Intn(500)
		time.Sleep(time.Duration(v) * time.Millisecond) // add some random delay
		if !sr.rumExchangeTestMode && sr.syncNetworkType == conn.PubSub {
			return connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
		} else {
			//send the request, will wait for the response
			return connMgr.SendReqTrxRex(trx)
		}
	} else if task.Type == ConsensusSync {
		consensusSynctask, ok := task.Meta.(ConsensusSyncTask)
		if !ok {
			gsyncer_log.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
			return fmt.Errorf("<%s> Unsupported task %s", sr.group.Item.GroupId, task.Id)
		}

		group_log.Debugf("<%s> TaskSender with ConsensusSync Task, SessionId <%s>", sr.group.Item.GroupId, consensusSynctask.SessionId)
		//create protobuf msg,should move to chaindata pkg
		consensusReq := &quorumpb.ConsensusReq{
			MyEpoch: sr.group.Item.Epoch,
		}

		cbytes, err := proto.Marshal(consensusReq)
		if err != nil {
			return err
		}

		consensusMsg := &quorumpb.ConsensusMsg{
			GroupId:      sr.group.Item.GroupId,
			SessionId:    consensusSynctask.SessionId,
			MsgType:      quorumpb.ConsensusType_REQ,
			Payload:      cbytes,
			SenderPubkey: sr.group.Item.UserSignPubkey,
			TimeStamp:    time.Now().UnixNano(),
		}

		bbytes, err := proto.Marshal(consensusMsg)
		if err != nil {
			return err
		}

		msgHash := localcrypto.Hash(bbytes)

		var signature []byte
		ks := localcrypto.GetKeystore()
		signature, err = ks.EthSignByKeyName(sr.group.Item.GroupId, msgHash, sr.group.ChainCtx.nodename)

		if err != nil {
			return err
		}

		if len(signature) == 0 {
			return fmt.Errorf("create signature failed")
		}

		//save hash and signature
		consensusMsg.MsgHash = msgHash
		consensusMsg.SenderSign = signature

		group_log.Debugf("<%s> Create ConsensusMsg done, sessionId <%s>", sr.group.Item.GroupId, consensusMsg.SessionId)

		connMgr, err := conn.GetConn().GetConnMgr(sr.group.Item.GroupId)
		if err != nil {
			return err
		}

		err = connMgr.SentConsensusMsgPubsub(consensusMsg, conn.ProducerChannel)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("<%s> Unsupported task type %s", sr.group.Item.GroupId, task.Id)
}

func (sr *SyncerRunner) ResultReceiver(result *SyncResult) (int64, error) {
	syncerrunner_log.Debugf("<%s> ResultReceiver called", sr.group.Item.GroupId)

	trxtaskresult, ok := result.Data.(*quorumpb.Trx)
	if ok {
		//v := rand.Intn(5) + 1
		//time.Sleep(time.Duration(v) * time.Second) // fake workload
		//try to save the result to db
		nextepoch, err := sr.group.ChainCtx.HandleReqBlockResp(trxtaskresult)
		if err != nil {
			if err == ErrSyncDone {
				syncerrunner_log.Debugf("<%s> SYNC done", sr.group.Item.GroupId)
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
	syncerrunner_log.Debugf("<%s> AddTrxToSyncerQueue called", sr.group.Item.GroupId)
	sr.resultserialid++
	resultid := strconv.FormatUint(uint64(sr.resultserialid), 10)
	result := &SyncResult{Id: resultid, Data: trx}
	if sr.Status == SYNCING_FORWARD || sr.Status == SYNCING_BACKWARD {
		sr.gsyncer.AddResult(result)
	}
}

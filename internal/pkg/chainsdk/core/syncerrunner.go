package chain

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var syncerrunner_log = logging.Logger("syncerrunner")
var RETRY_LIMIT = 30 //retry times
var REQ_BLOCKS = 10  //request 1 blocks each time

const (
	IDLE            = 1
	SYNCING_FORWARD = 2
	LOCAL_SYNCING   = 3
	CONSENSUS_SYNC  = 4
	SYNC_FAILED     = 5
	CLOSE           = 6
)

type SyncerRunner struct {
	groupId  string
	nodename string
	cdnIface def.ChainDataSyncIface
	gsyncer  *Gsyncer
	chainCtx *Chain
}

func NewSyncerRunner(groupId string, nodename string, cdnIface def.ChainDataSyncIface, chainCtx *Chain) *SyncerRunner {
	syncerrunner_log.Debugf("<%s> NewSyncerRunner called", groupId)
	sr := &SyncerRunner{}
	sr.groupId = groupId
	sr.nodename = nodename
	sr.cdnIface = cdnIface
	sr.chainCtx = chainCtx

	//create and initial Get Task Apis
	taskGenerators := make(map[TaskType]func(args ...interface{}) (*SyncTask, error))
	taskGenerators[GetEpoch] = sr.GetNextEpochTask
	taskGenerators[ConsensusSync] = sr.GetConsensusSyncTask

	gs := NewGsyncer(groupId, taskGenerators, sr.TaskSender)
	gs.SetRetryWithNext(false)
	sr.gsyncer = gs

	return sr
}

func (sr *SyncerRunner) GetCurrentSyncTask() (string, TaskType, uint, error) {
	syncerrunner_log.Debugf("<%s> GetCurrentSyncTask called", sr.groupId)
	return sr.gsyncer.GetCurrentTask()
}

// define how to get next task, for example, taskid+1
func (sr *SyncerRunner) GetNextEpochTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetEpochTask called", sr.groupId)
	nextEpoch := sr.cdnIface.GetCurrEpoch() + 1
	taskmeta := EpochSyncTask{Epoch: nextEpoch}
	taskid := strconv.FormatUint(uint64(nextEpoch), 10)
	return &SyncTask{TaskId: taskid, Type: GetEpoch, RetryCount: 0, Meta: taskmeta}, nil
}

func (sr *SyncerRunner) GetConsensusSyncTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetConsensusSyncTask called", sr.groupId)
	taskmate := ConsensusSyncTask{SessionId: uuid.NewString()}
	return &SyncTask{TaskId: taskmate.SessionId, Type: ConsensusSync, RetryCount: 0, Meta: taskmate}, nil
}

func (sr *SyncerRunner) Start() error {
	syncerrunner_log.Debugf("<%s> Start called", sr.groupId)

	var task *SyncTask
	var err error

	//producer try get consensus before start sync block
	if sr.chainCtx.isProducer() {
		groupMgr_log.Debugf("<%s> producer(owner) node try get consensus before sync", sr.groupId)
		task, err = sr.GetConsensusSyncTask()
		if err != nil {
			return err
		}
	} else {
		//user node start sync directly
		groupMgr_log.Debugf("<%s> user node start epoch (block) sync", sr.groupId)
		task, err = sr.GetNextEpochTask()
		if err != nil {
			return err
		}
	}

	//start syncer and add the first task
	sr.gsyncer.Start()
	sr.gsyncer.AddTask(task)
	return nil
}

func (sr *SyncerRunner) GetConsensus() (sessionId string, err error) {
	syncerrunner_log.Debugf("<%s> GetConsensus called", sr.groupId)
	if sr.chainCtx.isProducer() {
		task, err := sr.GetConsensusSyncTask()
		if err != nil {
			return "", err
		}
		sr.gsyncer.AddTask(task)
		return task.TaskId, nil
	} else {
		return "", fmt.Errorf("user node can not call get consensus")
	}
}

func (sr *SyncerRunner) Stop() {
	syncerrunner_log.Debugf("<%s> Stop called", sr.groupId)
	sr.gsyncer.Stop()
}

func (sr *SyncerRunner) TaskSender(task *SyncTask) error {
	syncerrunner_log.Debugf("<%s> TaskSender called", sr.groupId)

	if task.Type == GetEpoch {
		epochSyncTask, ok := task.Meta.(EpochSyncTask)
		if !ok {
			gsyncer_log.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
			return fmt.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
		}
		syncerrunner_log.Debugf("<%s> TaskSender with GetEpoch Task, Epoch <%d>", sr.groupId, epochSyncTask.Epoch)
		var trx *quorumpb.Trx
		var trxerr error

		trx, trxerr = sr.chainCtx.GetTrxFactory().GetReqBlocksTrx("", sr.groupId, epochSyncTask.Epoch, int64(REQ_BLOCKS))
		if trxerr != nil {
			return trxerr
		}

		connMgr, err := conn.GetConn().GetConnMgr(sr.groupId)
		if err != nil {
			return err
		}

		v := rand.Intn(500)
		time.Sleep(time.Duration(v) * time.Millisecond) // add some random delay

		return connMgr.SendReqTrxRex(trx)
	} else if task.Type == ConsensusSync {
		consensusSynctask, ok := task.Meta.(ConsensusSyncTask)
		if !ok {
			gsyncer_log.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
			return fmt.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
		}

		syncerrunner_log.Debugf("<%s> TaskSender with ConsensusSync Task, SessionId <%s>", sr.groupId, consensusSynctask.SessionId)
		//create protobuf msg,should move to chaindata pkg
		consensusReq := &quorumpb.ConsensusReq{
			MyEpoch: sr.cdnIface.GetCurrEpoch(),
		}

		cbytes, err := proto.Marshal(consensusReq)
		if err != nil {
			return err
		}

		consensusMsg := &quorumpb.ConsensusMsg{
			GroupId:      sr.groupId,
			SessionId:    consensusSynctask.SessionId,
			MsgType:      quorumpb.ConsensusType_REQ,
			Payload:      cbytes,
			SenderPubkey: sr.chainCtx.group.Item.UserSignPubkey,
			TimeStamp:    time.Now().UnixNano(),
		}

		bbytes, err := proto.Marshal(consensusMsg)
		if err != nil {
			return err
		}

		msgHash := localcrypto.Hash(bbytes)

		var signature []byte
		ks := localcrypto.GetKeystore()
		signature, err = ks.EthSignByKeyName(sr.groupId, msgHash, sr.nodename)

		if err != nil {
			return err
		}

		if len(signature) == 0 {
			return fmt.Errorf("create signature failed")
		}

		//save hash and signature
		consensusMsg.MsgHash = msgHash
		consensusMsg.SenderSign = signature

		connMgr, err := conn.GetConn().GetConnMgr(sr.groupId)
		if err != nil {
			return err
		}

		err = connMgr.BroadcastConsensusMsg(consensusMsg)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("<%s> Unsupported task type %s", sr.groupId, task.TaskId)
}

func (sr *SyncerRunner) UpdateGetEpochResult(taskId string, nextAction uint) {
	syncerrunner_log.Debugf("<%s> UpdateGetEpochResult called", sr.groupId)
	result := &SyncResult{TaskId: taskId, Type: GetEpoch, nextAction: SyncerAction(nextAction)}
	if sr.gsyncer.Status == SYNCING_FORWARD {
		sr.gsyncer.AddResult(result)
	}
}

func (sr *SyncerRunner) UpdateConsensusResult(taskId string, nextAction uint) {
	syncerrunner_log.Debugf("<%s> UpdateConsensusResult called", sr.gsyncer.GroupId)
	result := &SyncResult{TaskId: taskId, Type: ConsensusSync, nextAction: SyncerAction(nextAction)}
	if sr.gsyncer.Status == CONSENSUS_SYNC {
		sr.gsyncer.AddResult(result)
	}
}

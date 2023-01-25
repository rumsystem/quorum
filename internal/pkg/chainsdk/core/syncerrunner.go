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
	IDLE          = 1
	SYNCING_BLOCK = 2
	LOCAL_SYNCING = 3
	PSYNC         = 4
	SYNC_FAILED   = 5
	CLOSE         = 6
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
	taskGenerators[PSync] = sr.GetPSyncTask

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

func (sr *SyncerRunner) GetPSyncTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetConsensusSyncTask called", sr.groupId)
	taskmate := PSyncTask{SessionId: uuid.NewString()}
	return &SyncTask{TaskId: taskmate.SessionId, Type: PSync, RetryCount: 0, Meta: taskmate}, nil
}

func (sr *SyncerRunner) Start() error {
	syncerrunner_log.Debugf("<%s> Start called", sr.groupId)
	syncerrunner_log.Warning("!!!!!!!!!!!!!!!!!!!!!!!!! skip initial sync, commented by cuicat for test !!!!!!!!!!!!!!")
	/*
		var task *SyncTask
		var err error

		//producer try get consensus before start sync block
		if sr.chainCtx.isProducer() {
			syncerrunner_log.Debugf("<%s> producer(owner) node try get latest chain info before start sync", sr.groupId)


			task, err = sr.GetPSyncTask()
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

	*/
	return nil
}

func (sr *SyncerRunner) GetPSync() (sessionId string, err error) {
	syncerrunner_log.Debugf("<%s> GetPSync called", sr.groupId)
	if sr.chainCtx.isProducer() {
		task, err := sr.GetPSyncTask()
		if err != nil {
			return "", err
		}
		sr.gsyncer.AddTask(task)
		return task.TaskId, nil
	} else {
		return "", fmt.Errorf("user node can not call get psync")
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
	} else if task.Type == PSync {
		psynctask, ok := task.Meta.(PSyncTask)
		if !ok {
			gsyncer_log.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
			return fmt.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
		}

		syncerrunner_log.Debugf("<%s> TaskSender with PSync Task, SessionId <%s>", sr.groupId, psynctask.SessionId)

		//create psyncReqMsg
		psyncReqMsg := &quorumpb.PSyncReq{
			GroupId:      sr.groupId,
			SessionId:    psynctask.SessionId,
			SenderPubkey: sr.chainCtx.groupItem.UserSignPubkey,
			MyEpoch:      sr.chainCtx.GetCurrEpoch(),
		}

		//sign it
		bbytes, err := proto.Marshal(psyncReqMsg)
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

		psyncReqMsg.SenderSign = signature

		payload, _ := proto.Marshal(psyncReqMsg)
		psyncMsg := &quorumpb.PSyncMsg{
			MsgType: quorumpb.PSyncMsgType_PSYNC_REQ,
			Payload: payload,
		}

		connMgr, err := conn.GetConn().GetConnMgr(sr.groupId)
		if err != nil {
			return err
		}

		err = connMgr.BroadcastPSyncMsg(psyncMsg)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("<%s> Unsupported task type %s", sr.groupId, task.TaskId)
}

func (sr *SyncerRunner) UpdateGetEpochResult(taskId string, nextAction uint) {
	syncerrunner_log.Debugf("<%s> UpdateGetEpochResult called", sr.groupId)
	if sr.gsyncer.Status == SYNCING_BLOCK {
		result := &SyncResult{TaskId: taskId, Type: GetEpoch, nextAction: SyncerAction(nextAction)}
		sr.gsyncer.AddResult(result)
	}
}

func (sr *SyncerRunner) UpdatePSyncResult(taskId string, nextAction uint) {
	syncerrunner_log.Debugf("<%s> UpdatePSyncResult called", sr.gsyncer.GroupId)
	if sr.gsyncer.Status == PSYNC {
		result := &SyncResult{TaskId: taskId, Type: PSync, nextAction: SyncerAction(nextAction)}
		sr.gsyncer.AddResult(result)
	}
}

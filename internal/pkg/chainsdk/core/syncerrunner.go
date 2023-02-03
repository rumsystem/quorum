package chain

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var syncerrunner_log = logging.Logger("syncerrunner")

var RETRY_LIMIT = 30            //retry times
var REQ_BLOCKS = 10             //request 1 blocks each time
var SYNC_BLOCK_TASK_TIMEOUT = 4 //seconds

type TaskType uint

const (
	SYNC_TYPE_LOCAL TaskType = iota
	SYNC_TYPE_BLOCK
	SYNC_TYPE_CNSUS
)

type SyncSession struct {
	SessionId string
	Type      TaskType
	GSyncer   *GSyncer
}

type SyncBlockMeta struct {
	FromEpoch int64
	Request   int
}

type SyncCnsusMeta struct {
}

type SyncLocalMeta struct {
}

type SyncerRunner struct {
	groupId            string
	nodename           string
	cdnIface           def.ChainDataSyncIface
	chainCtx           *Chain
	SyncSessionsById   map[string]*SyncSession   //map[SessionID]
	SyncSessionsByType map[TaskType]*SyncSession //map[TaskType], can have only 1 task type at once
}

func NewSyncerRunner(groupId string, nodename string, cdnIface def.ChainDataSyncIface, chainCtx *Chain) *SyncerRunner {
	syncerrunner_log.Debugf("<%s> NewSyncerRunner called", groupId)
	sr := &SyncerRunner{}
	sr.groupId = groupId
	sr.nodename = nodename
	sr.cdnIface = cdnIface
	sr.chainCtx = chainCtx
	sr.SyncSessionsById = make(map[string]*SyncSession)
	sr.SyncSessionsByType = make(map[TaskType]*SyncSession)

	/*
		//create and initial Get Task Apis
		taskGenerators := make(map[TaskType]func(args ...interface{}) (*SyncTask, error))
		taskGenerators[GetEpoch] = sr.GetNextEpochTask
		taskGenerators[PSync] = sr.GetPSyncTask

		gs := NewGsyncer(groupId, taskGenerators, sr.TaskSender)
		sr.gsyncer = gs

	*/

	return sr
}

func (sr *SyncerRunner) StartBlockSync() error {
	syncerrunner_log.Debugf("<%s> StartBlockSync", sr.groupId)

	//check if other sync in ongoing
	if _, ok := sr.SyncSessionsByType[SYNC_TYPE_LOCAL]; ok {
		return fmt.Errorf("local sync ongoing, wait")
	}

	if _, ok := sr.SyncSessionsByType[SYNC_TYPE_BLOCK]; ok {
		return fmt.Errorf("another block syncing is running")
	}

	sessionId := uuid.NewString()
	syncerrunner_log.Debugf("<%s> start epoch (block) sync with sessionId <%s>", sr.groupId, sessionId)

	gsyncer := NewGsyncer(sr.groupId, sr.syncBlockTaskGenerator, sr.syncBlockTaskSender, sr.syncBlockMsgHandler, SYNC_BLOCK_TASK_TIMEOUT)

	//save current session
	session := &SyncSession{
		SessionId: sessionId,
		Type:      SYNC_TYPE_BLOCK,
		GSyncer:   gsyncer,
	}

	//create reference
	sr.SyncSessionsById[sessionId] = session
	sr.SyncSessionsByType[SYNC_TYPE_BLOCK] = session

	//start syncer and add the first task
	gsyncer.Start()
	gsyncer.Next()
	return nil
}

func (sr *SyncerRunner) Stop() {
	syncerrunner_log.Debugf("<%s> Stop called", sr.groupId)
	//shutdown all syncsession peacefully
	for _, session := range sr.SyncSessionsById {
		syncerrunner_log.Debugf("<%s> try stop session <%s>", sr.groupId, session.SessionId)
		session.GSyncer.Stop()
	}
}

// task generators
func (sr *SyncerRunner) syncBlockTaskGenerator(args ...interface{}) *SyncTask {
	syncerrunner_log.Debugf("<%s> GetEpochTask called", sr.groupId)
	nextEpoch := sr.cdnIface.GetCurrEpoch() + 1
	taskId := uuid.NewString()
	sessionId := args[0].(string)
	taskmeta := SyncBlockMeta{FromEpoch: nextEpoch, Request: REQ_BLOCKS}
	return &SyncTask{SessionId: sessionId, TaskId: taskId, RetryCount: 0, Meta: taskmeta}
}

func (sr *SyncerRunner) syncBlockTaskSender(task *SyncTask) error {
	syncerrunner_log.Debugf("<%s> syncBlockTaskSender called", sr.groupId)

	var trx *quorumpb.Trx
	var trxerr error

	syncmeta := task.Meta.(SyncBlockMeta)

	trx, trxerr = sr.chainCtx.GetTrxFactory().GetReqBlocksTrx("", sr.groupId, task.SessionId, syncmeta.FromEpoch, int64(syncmeta.Request))
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
}

func (sr *SyncerRunner) syncBlockMsgHandler(msg *SyncMsg) error {
	return nil
}

func (sr *SyncerRunner) HandleSyncMsg(sessionId string, msg *SyncMsg) error {
	if _, ok := sr.SyncSessionsById[sessionId]; !ok {
		return fmt.Errorf("can not find related sync session with id <%d>", sessionId)
	}
	sr.SyncSessionsById[sessionId].GSyncer.AddMsg(msg)
	return nil
}

/*
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


func (sr *SyncerRunner) GetSyncCnsusTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetConsensusSyncTask called", sr.groupId)
	taskmate := PSyncTask{SessionId: uuid.NewString()}
	return &SyncTask{TaskId: taskmate.SessionId, Type: PSync, RetryCount: 0, Meta: taskmate}, nil
}


func (sr *SyncerRunner) GetSyncLocalTask(args ...interface{}) (*SyncTask, error) {
	return nil, nil
}
*/

/*
else if task.Type == PSync {
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
*/

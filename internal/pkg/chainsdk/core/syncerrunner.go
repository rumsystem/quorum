package chain

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
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
	FromEpoch       int64
	RequestBlockNum int
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

	gsyncer := NewGsyncer(sr.groupId, sessionId, sr.SyncBlockTaskGenerator, sr.SyncBlockTaskSender, sr.SyncBlockMsgHandler, SYNC_BLOCK_TASK_TIMEOUT)

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
func (sr *SyncerRunner) SyncBlockTaskGenerator(args ...interface{}) *SyncTask {
	syncerrunner_log.Debugf("<%s> GetEpochTask called", sr.groupId)
	nextEpoch := sr.cdnIface.GetCurrEpoch() + 1
	taskId := uuid.NewString()
	sessionId := args[0].(string)
	taskmeta := SyncBlockMeta{FromEpoch: nextEpoch, RequestBlockNum: REQ_BLOCKS}
	return &SyncTask{SessionId: sessionId, TaskId: taskId, RetryCount: 0, Meta: taskmeta}
}

func (sr *SyncerRunner) SyncBlockTaskSender(task *SyncTask) error {
	syncerrunner_log.Debugf("<%s> syncBlockTaskSender called", sr.groupId)

	var trx *quorumpb.Trx
	var trxerr error

	syncmeta := task.Meta.(SyncBlockMeta)

	trx, trxerr = sr.chainCtx.GetTrxFactory().GetReqBlocksTrx("", sr.groupId, task.SessionId, task.TaskId, syncmeta.FromEpoch, int64(syncmeta.RequestBlockNum))
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

func (sr *SyncerRunner) SyncBlockMsgHandler(msg *SyncMsg, gsyncer *GSyncer) error {
	syncerrunner_log.Debugf("<%s> SyncBlockMsgHandler called", sr.groupId)
	reqBlockResp := msg.Data.(*quorumpb.ReqBlockResp)

	//if not asked by me, ignore it
	if reqBlockResp.RequesterPubkey != sr.chainCtx.groupItem.UserSignPubkey {
		//chain_log.Debugf("<%s> HandleReqBlockResp error <%s>", chain.Group.GroupId, rumerrors.ErrSenderMismatch.Error())
		return rumerrors.ErrNotAskedByMe
	}

	//check if the resp is what we are waiting for
	if reqBlockResp.TaskId != gsyncer.CurrentTask.TaskId {
		//chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.groupItem.GroupId, rumerrors.ErrEpochMismatch)
		return rumerrors.ErrTaskIdMismatch
	}

	syncerrunner_log.Debugf("- Receive valid reqBlockResp, provider <%s> result <%s> from epoch <%d> total blocks provided <%d>",
		reqBlockResp.ProviderPubkey,
		reqBlockResp.Result.String(),
		reqBlockResp.FromEpoch,
		len(reqBlockResp.Blocks.Blocks))

	//Since a valid response is retrieved, finish current task anyway
	gsyncer.CurrentTaskDone()

	//isFromProducer := chain.isProducerByPubkey(reqBlockResp.ProviderPubkey)
	// since only 1 producer (owner) is supported in this version
	// node should only accept BLOCK_NOT_FOUND from group owner
	isOwner := sr.chainCtx.isOwnerByPubkey(reqBlockResp.ProviderPubkey)

	switch reqBlockResp.Result {
	case quorumpb.ReqBlkResult_BLOCK_NOT_FOUND:
		/*
			//user node say BLOCK_NOT_FOUND, ignore
			if !isFromProducer {
				chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_NOT_FOUND from user node <%s>, ignore", chain.groupItem.GroupId, reqBlockResp.ProviderPubkey)
				return
			}

			//TBD, stop only when received BLOCK_NOT_FOUND from F + 1 producers, otherwise continue sync
			chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_NOT_FOUND from producer node <%s>, process it", chain.groupItem.GroupId, reqBlockResp.ProviderPubkey)
		*/
		// since only 1 producer (owner) is supported in this version
		// node should only accept BLOCK_NOT_FOUND from group owner
		// and ignore all other BLOCK_NOT_FOUND msg
		if isOwner {
			chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_NOT_FOUND from group owner, stop sync", sr.groupId)
			gsyncer.Stop()
			//remove syncer and reference from
			delete(sr.SyncSessionsById, gsyncer.SessionId)
			delete(sr.SyncSessionsByType, SYNC_TYPE_BLOCK)

			return nil
		}

		gsyncer.Next()

	case quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP:
		sr.chainCtx.ApplyBlocks(reqBlockResp.Blocks.Blocks)
		/*

			if !isFromProducer {
				chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP_ON_TOP from user node <%s>, apply all blocks and  ignore ON_TOP", chain.groupItem.GroupId, reqBlockResp.ProviderPubkey)
				return
			}

			chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP_ON_TOP from producer node <%s>, process it", chain.groupItem.GroupId, reqBlockResp.ProviderPubkey)
			//ignore on_top msg, run another round of sync, till get F + 1 BLOCK_NOT_FOUND from producers

		*/

		if isOwner {
			chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP_ON_TOP from group owner, apply blocks and stop sync", sr.groupId)
			gsyncer.Stop()
			//remove syncer and reference from
			delete(sr.SyncSessionsById, gsyncer.SessionId)
			delete(sr.SyncSessionsByType, SYNC_TYPE_BLOCK)

			return nil
		}

		gsyncer.Next()
		return nil
	case quorumpb.ReqBlkResult_BLOCK_IN_RESP:
		chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP from node <%s>, apply all blocks", sr.groupId, reqBlockResp.ProviderPubkey)
		sr.chainCtx.ApplyBlocks(reqBlockResp.Blocks.Blocks)
		gsyncer.Next()
		return nil
	default:
		//do nothing
	}

	return nil
}

func (sr *SyncerRunner) HandleSyncResp(trx *quorumpb.Trx, typ TaskType) error {
	syncerrunner_log.Debugf("<%s> HandleSyncResp called", sr.groupId)
	//decode resp
	var err error
	ciperKey, err := hex.DecodeString(sr.chainCtx.chaindata.groupCipherKey)
	if err != nil {
		syncerrunner_log.Warningf("<%s> HandleSyncResp error <%s>", sr.groupId, err.Error())
		return err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		syncerrunner_log.Warningf("<%s> HandleSyncResp error <%s>", sr.groupId, err.Error())
		return err
	}

	switch typ {
	case SYNC_TYPE_BLOCK:
		reqBlockResp := &quorumpb.ReqBlockResp{}
		if err := proto.Unmarshal(decryptData, reqBlockResp); err != nil {
			chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", sr.groupId, err.Error())
			return err
		}

		if _, ok := sr.SyncSessionsById[reqBlockResp.SessionId]; !ok {
			return fmt.Errorf("can not find related sync session with id <%s>", reqBlockResp.SessionId)
		}

		syncmsg := &SyncMsg{
			TaskId: reqBlockResp.TaskId,
			Data:   reqBlockResp,
		}

		sr.SyncSessionsById[reqBlockResp.SessionId].GSyncer.AddMsg(syncmsg)
		return nil

	case SYNC_TYPE_CNSUS:
		//case it to consusSync type
		syncerrunner_log.Debugf("<%s> HandleSyncResp CNSUS, TBD", sr.groupId)
	default:
		chain_log.Debugf("<%s> Unknown SyncResp type, ignore", sr.groupId)
		return fmt.Errorf("unknown syncresp type")
	}

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

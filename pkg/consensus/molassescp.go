package consensus

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var molacp_log = logging.Logger("cp")

type ConsensusProposeTask struct {
	TrxId    string
	Req      *quorumpb.ChangeConsensusReq
	ReqBytes []byte
	ctx      context.Context
	cancel   context.CancelFunc
}

type BftTask struct {
	bft       *PCBft
	chBftDone chan *quorumpb.ChangeConsensusResultBundle
	Proof     *quorumpb.ConsensusProof
	ctx       context.Context
	cancel    context.CancelFunc
}

type MolassesConsensusProposer struct {
	grpItem  *quorumpb.GroupItem
	groupId  string
	nodename string
	cIface   def.ChainMolassesIface
	chainCtx context.Context

	chProposeTask chan *ConsensusProposeTask
	chBftTask     chan *BftTask

	currCpTask  *ConsensusProposeTask
	currBftTask *BftTask

	lock sync.Mutex
}

func (cp *MolassesConsensusProposer) NewConsensusProposer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molacp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
	cp.grpItem = item
	cp.groupId = item.GroupId
	cp.nodename = nodename
	cp.cIface = iface
	cp.chainCtx = ctx

	cp.chProposeTask = make(chan *ConsensusProposeTask, 1)
	cp.chBftTask = make(chan *BftTask, 1)

	go cp.ProposeWorker(ctx, cp.chProposeTask)
	go cp.BftWorker(ctx, cp.chBftTask)

	cp.currCpTask = nil
	cp.currBftTask = nil
}

func (cp *MolassesConsensusProposer) ProposeWorker(chainCtx context.Context, chProposeTask <-chan *ConsensusProposeTask) {
	for {
		select {
		case <-chainCtx.Done():
			molacp_log.Debugf("<%s> ProposerWorker exit", cp.groupId)
			return
		case task, beforeChClosed := <-chProposeTask:
			if chainCtx.Err() != nil {
				molacp_log.Debugf("<%s> ProposerWorker: chainCtx canceled, return", cp.groupId)
				return
			}

			if !beforeChClosed {
				molacp_log.Debugf("<%s> ProposerWorker: channel closed, return", cp.groupId)
				return
			}

			retryCnt := 0
			isCanceled := false
			//handle it
			var err error
		RETRY:

			for retryCnt < int(task.Req.AgreementTickCount) {
				reqMsg := &quorumpb.ChangeConsensusReqMsg{
					Req:   task.Req,
					Epoch: uint64(retryCnt),
				}

				molacp_log.Debugf("<%s> change consensus ROUND <%d> req <%s>", cp.groupId, retryCnt, task.Req.ReqId)
				connMgr, err := conn.GetConn().GetConnMgr(cp.groupId)
				if err != nil {
					molacp_log.Errorf("<%s> ProposerWorker: GetConnMgr failed, err <%v>", cp.groupId, err)
					break RETRY
				}
				connMgr.BroadcastCCReqMsg(reqMsg)
				select {
				case <-task.ctx.Done():
					molacp_log.Debugf("<%s> ProposerWorker: taskCtx done", cp.groupId)
					isCanceled = true
					break RETRY
				case <-time.After(time.Duration(task.Req.AgreementTickLenInMs) * time.Millisecond):
					molacp_log.Debugf("<%s> change consensus ROUND <%d> timeout", cp.groupId, retryCnt)
					retryCnt += 1
				}
			}

			if err != nil {
				molacp_log.Errorf("<%s> ProposerWorker: change consensus failed, err <%v>", cp.groupId, err)
				resultBundle := &quorumpb.ChangeConsensusResultBundle{
					Result:             quorumpb.ChangeConsensusResult_FAIL,
					Req:                task.Req,
					Resps:              nil,
					ResponsedProducers: nil,
				}
				cp.cIface.ChangeConsensusDone(resultBundle, cp.currCpTask.TrxId)

			} else {
				if isCanceled {
					molacp_log.Debugf("<%s> ProposerWorker: taskCtx canceled", cp.groupId)
				} else {
					//if goes here, means no consensus reached and timeout, notify chain and quit
					molacp_log.Debugf("<%s> ProposerWorker: timeout and no consensus reached, notify chain", cp.groupId)
					resultBundle := &quorumpb.ChangeConsensusResultBundle{
						Result:             quorumpb.ChangeConsensusResult_TIMEOUT,
						Req:                task.Req,
						Resps:              nil,
						ResponsedProducers: nil,
					}
					cp.cIface.ChangeConsensusDone(resultBundle, cp.currCpTask.TrxId)
				}
			}

			molacp_log.Debugf("<%s> ProposerWorker: task done", cp.groupId)
		}
	}
}

func (cp *MolassesConsensusProposer) BftWorker(chainCtx context.Context, chBftTask <-chan *BftTask) {
	for {
		select {
		case <-chainCtx.Done():
			molacp_log.Debugf("<%s> BftWorker exit", cp.groupId)
			return
		case task, beforeChClosed := <-chBftTask:
			//handle it
			if chainCtx.Err() != nil {
				molacp_log.Debugf("<%s> BftWorker: chainCtx canceled, return", cp.groupId)
				return
			}

			if !beforeChClosed {
				molacp_log.Debugf("<%s> BftWorker: channel closed, return", cp.groupId)
				return
			}

			molacp_log.Debugf("<%s> BftWorker: start bft", cp.groupId)
			err := task.bft.Propose()

			isCanceled := false
			select {
			case <-task.ctx.Done():
				molacp_log.Debugf("<%s> BftWorker, taskCtx done, quit task", cp.groupId)
				isCanceled = true
			case result := <-task.chBftDone:
				molacp_log.Debugf("<%s> HandleCCReq bft done with result", cp.groupId)
				if cp.currCpTask != nil {
					cp.cIface.ChangeConsensusDone(result, cp.currCpTask.TrxId)
				} else {
					cp.cIface.ChangeConsensusDone(result, "")
				}
			}

			if err != nil {
				molacp_log.Errorf("<%s> BftWorker: bftTask failed, err <%s>", cp.groupId, err.Error())
			} else {
				if isCanceled {
					molacp_log.Debugf("<%s> BftWorker: bftTask done (canceled)", cp.groupId)
				} else {
					molacp_log.Debugf("<%s> BftWorker: bftTask done", cp.groupId)
				}
			}
		}
	}
}

func (cp *MolassesConsensusProposer) StartChangeConsensus(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartChangeConsensus called", cp.groupId)

	cp.lock.Lock()
	defer cp.lock.Unlock()

	cpTask, err := cp.createProposeTask(cp.chainCtx, producers, trxId, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen)
	if err != nil {
		molacp_log.Errorf("<%s> createProposeTask failed", cp.groupId)
		return err
	}

	if cp.currCpTask != nil {
		cp.currCpTask.cancel()
	}

	cp.chProposeTask <- cpTask
	cp.currCpTask = cpTask

	return nil
}

func (cp *MolassesConsensusProposer) StopAllTasks() {
	molacp_log.Debugf("<%s> StopAllTasks called", cp.groupId)

	cp.lock.Lock()
	defer cp.lock.Unlock()

	if cp.currBftTask != nil {
		cp.currBftTask.cancel()
	}

	if cp.currCpTask != nil {
		cp.currCpTask.cancel()
	}

	cp.currBftTask = nil
	cp.currCpTask = nil

	molacp_log.Debugf("<%s> StopAllTasks done", cp.groupId)
}

func (cp *MolassesConsensusProposer) HandleCCReq(msg *quorumpb.ChangeConsensusReqMsg) error {
	molacp_log.Debugf("<%s> HandleCCReq called reqId <%s>, epoch <%d>", cp.groupId, msg.Req.ReqId, msg.Epoch)
	cp.lock.Lock()
	defer cp.lock.Unlock()

	bftTask, err := cp.createBftTask(cp.chainCtx, msg)
	if err != nil {
		molacp_log.Debugf("<%s> HandleCCReq create bft task failed with error <%s>", cp.groupId, err.Error())
		return err
	} else {
		molacp_log.Debugf("<%s> HandleCCReq create bft task", cp.groupId)
	}

	if cp.currBftTask != nil {
		molacp_log.Debugf("<%s> HandleCCReq: currBftTask is not nil, cancel it", cp.groupId)
		cp.currBftTask.cancel()
	}

	cp.chBftTask <- bftTask
	cp.currBftTask = bftTask

	return nil
}

func (cp *MolassesConsensusProposer) createProposeTask(ctx context.Context, producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) (*ConsensusProposeTask, error) {
	molacp_log.Debugf("<%s> createProposeTask called", cp.groupId)
	nonce, err := nodectx.GetNodeCtx().GetChainStorage().GetNextConsensusNonce(cp.groupId, cp.nodename)
	if err != nil {
		molacp_log.Errorf("<%s> get next consensus nonce failed", cp.groupId)
		return nil, err
	}

	//skip nonce 0 since nonce 0 is used by owner when create the group
	if nonce == 0 {
		nonce, err = nodectx.GetNodeCtx().GetChainStorage().GetNextConsensusNonce(cp.groupId, cp.nodename)
		if err != nil {
			molacp_log.Errorf("<%s> get next consensus nonce failed", cp.groupId)
			return nil, err
		}
	}

	molacp_log.Debugf("<%s> get next consensus nonce <%d> ", cp.groupId, nonce)

	for _, p := range producers {
		molacp_log.Debugf("<%s> producer <%s>", cp.groupId, p)
	}

	//create req
	req := &quorumpb.ChangeConsensusReq{
		ReqId:                guuid.New().String(),
		GroupId:              cp.groupId,
		Nonce:                nonce,
		ProducerPubkeyList:   producers,
		AgreementTickLenInMs: agrmTickLen,
		AgreementTickCount:   agrmTickCnt,
		StartFromEpoch:       fromNewEpoch,
		TrxEpochTickLenInMs:  trxEpochTickLen,
		SenderPubkey:         cp.grpItem.UserSignPubkey,
	}

	byts, err := proto.Marshal(req)
	if err != nil {
		molacp_log.Errorf("<%s> marshal change consensus req failed", cp.groupId)
		return nil, err
	}

	hash := localcrypto.Hash(byts)
	ks := nodectx.GetNodeCtx().Keystore
	signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
	req.MsgHash = hash
	req.SenderSign = signature

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		molacp_log.Errorf("<%s> marshal change consensus req failed", cp.groupId)
		return nil, err
	}

	cpTask := &ConsensusProposeTask{
		TrxId:    trxId,
		Req:      req,
		ReqBytes: reqBytes,
	}

	//create sender ctx
	cpTask.ctx, cpTask.cancel = context.WithCancel(cp.chainCtx)
	return cpTask, nil
}

func (cp *MolassesConsensusProposer) createBftTask(ctx context.Context, msg *quorumpb.ChangeConsensusReqMsg) (*BftTask, error) {
	molacp_log.Debugf("<%s> createBftTask called", cp.groupId)

	//check if req is from group owner
	if cp.grpItem.OwnerPubKey != msg.Req.SenderPubkey {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is not from group owner, ignore", cp.groupId, msg.Req.ReqId)
		return nil, fmt.Errorf("req is not from group owner")
	}

	//check if I am in the producer list (not owner)
	if !cp.cIface.IsOwner() {
		inTheList := false
		for _, pubkey := range msg.Req.ProducerPubkeyList {
			if pubkey == cp.grpItem.UserSignPubkey {
				inTheList = true
				break
			}
		}
		if !inTheList {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is not for me, ignore", cp.groupId, msg.Req.ReqId)
			return nil, fmt.Errorf("req is not for me")
		}
	}
	//verify if req is valid
	dumpreq := &quorumpb.ChangeConsensusReq{
		ReqId:                msg.Req.ReqId,
		GroupId:              msg.Req.GroupId,
		Nonce:                msg.Req.Nonce,
		ProducerPubkeyList:   msg.Req.ProducerPubkeyList,
		AgreementTickLenInMs: msg.Req.AgreementTickLenInMs,
		AgreementTickCount:   msg.Req.AgreementTickCount,
		StartFromEpoch:       msg.Req.StartFromEpoch,
		TrxEpochTickLenInMs:  msg.Req.TrxEpochTickLenInMs,
		SenderPubkey:         msg.Req.SenderPubkey,
		MsgHash:              nil,
		SenderSign:           nil,
	}

	byts, err := proto.Marshal(dumpreq)
	if err != nil {
		molacp_log.Errorf("<%s> marshal change consensus req failed", cp.groupId)
		return nil, err
	}

	hash := localcrypto.Hash(byts)
	if !bytes.Equal(hash, msg.Req.MsgHash) {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> hash is not same as req.MsgHash, ignore", cp.groupId, msg.Req.ReqId)
		return nil, fmt.Errorf("hash is not same as req.MsgHash")
	}

	//verify signature
	verifySign, err := cp.cIface.VerifySign(hash, msg.Req.SenderSign, msg.Req.SenderPubkey)

	if err != nil {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> failed with error <%s>", cp.groupId, msg.Req.ReqId, err.Error())
		return nil, err
	}

	if !verifySign {
		molacp_log.Debug("<%s> HandleCCReq reqid <%s> verify sign failed", cp.groupId, msg.Req.ReqId)
		return nil, fmt.Errorf("verify sign failed")
	}

	//TBD: check if nonce is valid (should large than current nonce)

	// create resp
	resp := &quorumpb.ChangeConsensusResp{
		RespId:       guuid.New().String(),
		GroupId:      msg.Req.GroupId,
		SenderPubkey: cp.grpItem.UserSignPubkey,
		Req:          msg.Req,
		MsgHash:      nil,
		SenderSign:   nil,
	}

	byts, err = proto.Marshal(resp)
	if err != nil {
		molacp_log.Errorf("<%s> marshal change consensus resp failed", cp.groupId)
		return nil, err
	}

	hash = localcrypto.Hash(byts)
	resp.MsgHash = hash
	ks := nodectx.GetNodeCtx().Keystore
	signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
	resp.SenderSign = signature

	// create Proof
	proofBundle := &quorumpb.ConsensusProof{
		Epoch: msg.Epoch,
		Req:   msg.Req,
		Resp:  resp,
	}

	participants := make([]string, 0)
	participants = append(participants, msg.Req.ProducerPubkeyList...)

	isOwnerInProducerList := false
	for _, producer := range msg.Req.ProducerPubkeyList {
		if producer == cp.grpItem.OwnerPubKey {
			isOwnerInProducerList = true
			break
		}
	}

	//if not add owner to the list to finish consensus
	if !isOwnerInProducerList {
		molacp_log.Debugf("<%s> owner is not in the producer list, to make consensus finished add owner to participants list", cp.groupId)
		participants = append(participants, cp.grpItem.OwnerPubKey)
	} else {
		molacp_log.Debugf("<%s> owner is in the producer list", cp.groupId)
	}

	// create bft config
	config, err := cp.createBftConfig(participants)
	if err != nil {
		molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
		return nil, err
	}

	// create new context
	bftCtx, bftCancel := context.WithCancel(cp.chainCtx)

	// create channel to receive bft result
	chBftDone := make(chan *quorumpb.ChangeConsensusResultBundle, 1)

	// create bft task
	bftTask := &BftTask{
		ctx:       bftCtx,
		cancel:    bftCancel,
		bft:       NewPCBft(bftCtx, *config, chBftDone, cp.cIface),
		chBftDone: chBftDone,
		Proof:     proofBundle,
	}

	bftTask.bft.AddProof(proofBundle)
	return bftTask, nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)
	if cp.currBftTask != nil {
		cp.currBftTask.bft.HandleHBMsg(hbmsg)
	}
	return nil
}

func (cp *MolassesConsensusProposer) createBftConfig(producers []string) (*Config, error) {
	molacp_log.Debugf("<%s> createBftConfig called", cp.groupId)

	config := &Config{
		GroupId:     cp.groupId,
		NodeName:    cp.nodename,
		MyPubkey:    cp.grpItem.UserSignPubkey,
		OwnerPubKey: cp.grpItem.OwnerPubKey,

		N:         len(producers),
		f:         0,
		Nodes:     producers,
		BatchSize: 1,
	}

	return config, nil
}

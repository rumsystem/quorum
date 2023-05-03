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
	GOID             int
	TrxId            string
	BroadcastCnt     int
	senderCtx        context.Context
	senderCancelFunc context.CancelFunc
	Req              *quorumpb.ChangeConsensusReq
	ReqBytes         []byte
}

type BftTask struct {
	GOID          int
	bftCtx        context.Context
	bftCancelFunc context.CancelFunc
	bft           *PCBft
	chBftDone     chan *quorumpb.ChangeConsensusResultBundle
	Proof         *quorumpb.ConsensusProof
}

type MolassesConsensusProposer struct {
	grpItem  *quorumpb.GroupItem
	groupId  string
	nodename string
	cIface   def.ChainMolassesIface
	chainCtx context.Context

	chProposeTask chan *ConsensusProposeTask
	chBftTask     chan *BftTask

	chCpTaskCancelDone  chan bool
	chBftTaskCancelDone chan bool

	lock sync.Mutex
}

func (cp *MolassesConsensusProposer) ProposerWorker(ctx context.Context, chProposeTask <-chan *ConsensusProposeTask) {
	for {
		select {
		case <-ctx.Done():
			molacp_log.Debugf("<%s> ProposerWorker exit", cp.groupId)
			return
		case task := <-chProposeTask:			
			//handle it
			for cp.cpTask.BroadcastCnt < int(cp.cpTask.Req.AgreementTickCount) {
				reqMsg := &quorumpb.ChangeConsensusReqMsg{
					Req:   cp.cpTask.Req,
					Epoch: uint64(cp.cpTask.BroadcastCnt),
				}
	
				molacp_log.Debugf("<%s> change consensus ROUND <%d> req <%s>", cp.groupId, cp.cpTask.BroadcastCnt, cp.cpTask.Req.ReqId)
				connMgr, err := conn.GetConn().GetConnMgr(cp.groupId)
				if err != nil {
					return
				}
				connMgr.BroadcastCCReqMsg(reqMsg)
				select {
				case <-time.After(time.Duration(cp.cpTask.Req.AgreementTickLenInMs) * time.Millisecond):
					molacp_log.Debugf("<%s> change consensus ROUND <%d> timeout", cp.groupId, cp.cpTask.BroadcastCnt)
					cp.cpTask.BroadcastCnt += 1
				}
			}
			//if goes here, means no consensus reached and timeout, notify chain
			molacp_log.Debugf("<%s> no consensus reached, notify chain", cp.groupId)

			if ctx.Err() != nil {
				molacp_log.Debugf("<%s> ProposerWorker: context canceled, return", cp.groupId)
				return
			}		
		}	
	}
}

func (cp *MolassesConsensusProposer) BftWorker(ctx context.Context, chBftTask <-chan *BftTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-chBftTask:
			//handle it

			 if ctx.Err() != nil {
				molacp_log.Debugf("<%s> BftWorker: context canceled, return", cp.groupId)
				return
			 }
	}
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



	go cp.ProposerWorker(cp.chProposeTask)
	go cp.BftWorker(cp.chBftTask)

	cp.chCpTaskCancelDone = make(chan bool)
	cp.chBftTaskCancelDone = make(chan bool)
}

func (cp *MolassesConsensusProposer) StartChangeConsensus(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartChangeConsensus called", cp.groupId)

	if cp.cpTask != nil {
		cp.cpTask.senderCancelFunc()
		molacp_log.Debugf("<%s> wait for cpTask cancel done", cp.groupId)
		<-cp.chCpTaskCancelDone
	}
	cp.cpTask = nil

	if cp.bftTask != nil {
		cp.bftTask.bftCancelFunc()
		molacp_log.Debugf("<%s> wait for bftTask cancel done", cp.groupId)
		<-cp.chBftTaskCancelDone
	}
	cp.bftTask = nil

	cpTask, err := cp.createProposeTask(producers, trxId, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen, cp.chCpTaskCancelDone)
	if err != nil {
		molacp_log.Errorf("<%s> createProposeTask failed", cp.groupId)
		return err
	}

	cp.cpTask = cpTask

	go func() {
		
		//TBD: notify chain
	}()

	return nil
}

func (cp *MolassesConsensusProposer) StopAllTasks() {
	molacp_log.Debugf("<%s> StopAllTasks called", cp.groupId)
	if cp.cpTask != nil {
		cp.cpTask.senderCancelFunc()
		<-cp.chCpTaskCancelDone
	}
	if cp.bftTask != nil {
		cp.bftTask.bftCancelFunc()
		<-cp.chBftTaskCancelDone
	}
	cp.cpTask = nil
	cp.bftTask = nil

	molacp_log.Debugf("<%s> StopAllTasks done", cp.groupId)
}

func (cp *MolassesConsensusProposer) HandleCCReq(msg *quorumpb.ChangeConsensusReqMsg) error {
	molacp_log.Debugf("<%s> HandleCCReq called reqId <%s>, epoch <%d>", cp.groupId, msg.Req.ReqId, msg.Epoch)

	if cp.bftTask != nil {
		cp.bftTask.bftCancelFunc()
		<-cp.bftTask.cancelDone
		cp.bftTask = nil
	}

	bftTask, err := cp.createBftTask(msg, cp.chBftTaskCancelDone)
	if err != nil {
		molacp_log.Debugf("<%s> HandleCCReq create bft task failed with error <%s>", cp.groupId, err.Error())
		return err
	} else {
		molacp_log.Debugf("<%s> HandleCCReq create bft task <%d> success", cp.groupId, bftTask.GOID)
	}

	cp.bftTask = bftTask

	go func() {
		cp.bftTask.bft.Propose()
		select {
		case <-cp.bftTask.bftCtx.Done():
			molacp_log.Debugf("<%s> HandleCCReq bft context done, GOID <%d> ", cp.groupId, cp.bftTask.GOID)
			cp.bftTask.cancelDone <- true
			return
		case result := <-cp.bftTask.chBftDone:
			molacp_log.Debugf("<%s> HandleCCReq bft done with result", cp.groupId)
			if cp.cpTask != nil {
				cp.cIface.ChangeConsensusDone(result, cp.cpTask.TrxId)
			} else {
				cp.cIface.ChangeConsensusDone(result, "")
			}
			return
		}
	}()

	return nil
}

func (cp *MolassesConsensusProposer) createProposeTask(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64, cancelDone chan bool) (*ConsensusProposeTask, error) {
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
		GOID:         goid(),
		TrxId:        trxId,
		BroadcastCnt: 0,
		cancelDone:   cancelDone,
		Req:          req,
		ReqBytes:     reqBytes,
	}

	//create sender ctx
	cpTask.senderCtx, cpTask.senderCancelFunc = context.WithCancel(cp.chainCtx)
	return cpTask, nil
}

func (cp *MolassesConsensusProposer) createBftTask(msg *quorumpb.ChangeConsensusReqMsg, cancelDone chan bool) (*BftTask, error) {
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
	ks := nodectx.GetNodeCtx().Keystore
	signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
	resp.MsgHash = hash
	resp.SenderSign = signature

	// create Proof
	proofBundle := &quorumpb.ConsensusProof{
		Epoch: msg.Epoch,
		Req:   msg.Req,
		Resp:  resp,
	}

	// check if owner is in the producer list (if I am the owner)
	if cp.cIface.IsOwner() {
		isInProducerList := false
		for _, producer := range msg.Req.ProducerPubkeyList {
			if producer == cp.grpItem.UserSignPubkey {
				isInProducerList = true
				break
			}
		}

		//if not add owner to the list to finish consensus
		if !isInProducerList {
			molacp_log.Debugf("<%s> owner is not in the producer list, to make consensus finished add owner to producer list", cp.groupId)
			msg.Req.ProducerPubkeyList = append(msg.Req.ProducerPubkeyList, cp.grpItem.OwnerPubKey)
		}
	}

	// create bft config
	config, err := cp.createBftConfig(msg.Req.ProducerPubkeyList)
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
		GOID:          goid(),
		bftCtx:        bftCtx,
		bftCancelFunc: bftCancel,
		bft:           NewPCBft(bftCtx, *config, chBftDone, cp.cIface),
		chBftDone:     chBftDone,
		Proof:         proofBundle,
		cancelDone:    cancelDone,
	}

	bftTask.bft.AddProof(proofBundle)
	return bftTask, nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)
	if cp.bftTask != nil {
		cp.bftTask.bft.HandleHBMsg(hbmsg)
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

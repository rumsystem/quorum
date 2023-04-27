package consensus

import (
	"bytes"
	"context"
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

	currBftTask  *BftTask
	currSendTask *ConsensusProposeTask

	locker sync.Mutex
}

func (cp *MolassesConsensusProposer) NewConsensusProposer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molacp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
	cp.grpItem = item
	cp.groupId = item.GroupId
	cp.nodename = nodename
	cp.cIface = iface
	cp.chainCtx = ctx
}

func (cp *MolassesConsensusProposer) StartChangeConsensus(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartChangeConsensus called", cp.groupId)
	cp.locker.Lock()
	defer cp.locker.Unlock()

	if cp.currSendTask != nil {
		molacp_log.Debugf("<%s> previous send task is not finished, cancel it", cp.groupId)
		if cp.currSendTask.senderCancelFunc != nil {
			cp.currSendTask.senderCancelFunc()
		}
	}

	if cp.currBftTask != nil {
		molacp_log.Debugf("<%s> previous bft task is not finished, cancel it", cp.groupId)
		if cp.currBftTask.bftCancelFunc != nil {
			cp.currBftTask.bftCancelFunc()
		}
	}

	cp.currSendTask = nil
	cp.currBftTask = nil

	go func() {
		cp.currSendTask = &ConsensusProposeTask{
			GOID:         goid(),
			TrxId:        trxId,
			BroadcastCnt: 0,
		}

		molacp_log.Debugf("<%s> start sender task, GOID <%d>", cp.groupId, cp.currSendTask.GOID)

		nonce, err := nodectx.GetNodeCtx().GetChainStorage().GetNextConsensusNonce(cp.groupId, cp.nodename)
		if err != nil {
			molacp_log.Errorf("<%s> get next consensus nonce failed", cp.groupId)
			return
		}

		//nonce 0 is used by owner when create the group
		if nonce == 0 {
			nonce, err = nodectx.GetNodeCtx().GetChainStorage().GetNextConsensusNonce(cp.groupId, cp.nodename)
			if err != nil {
				molacp_log.Errorf("<%s> get next consensus nonce failed", cp.groupId)
				return
			}
		}

		molacp_log.Debugf("<%s> get consensus nonce done <%d> ", cp.groupId, nonce)

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
			return
		}

		hash := localcrypto.Hash(byts)
		ks := nodectx.GetNodeCtx().Keystore
		signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
		req.MsgHash = hash
		req.SenderSign = signature

		cp.currSendTask.Req = req
		signedbyts, err := proto.Marshal(req)
		if err != nil {
			molacp_log.Errorf("<%s> marshal change consensus req failed", cp.groupId)
			return
		}
		cp.currSendTask.ReqBytes = signedbyts
		cp.currSendTask.senderCtx, cp.currSendTask.senderCancelFunc = context.WithCancel(cp.chainCtx)

		for cp.currSendTask.BroadcastCnt < int(cp.currSendTask.Req.AgreementTickCount) {
			reqMsg := &quorumpb.ChangeConsensusReqMsg{
				Req:   cp.currSendTask.Req,
				Epoch: uint64(cp.currSendTask.BroadcastCnt),
			}

			molacp_log.Debugf("<%s> change consensus ROUND <%d> req <%s>", cp.groupId, cp.currSendTask.BroadcastCnt, cp.currSendTask.Req.ReqId)
			connMgr, err := conn.GetConn().GetConnMgr(cp.groupId)
			if err != nil {
				return
			}
			connMgr.BroadcastCCReqMsg(reqMsg)

			select {
			case <-cp.currSendTask.senderCtx.Done():
				molacp_log.Debugf("<%s> sendTask ctx Done, GOID <%d> ", cp.groupId, cp.currSendTask.GOID)
				return
			case <-time.After(time.Duration(cp.currSendTask.Req.AgreementTickLenInMs) * time.Millisecond):
				molacp_log.Debugf("<%s> change consensus ROUND <%d> timeout", cp.groupId, cp.currSendTask.BroadcastCnt)
				cp.currSendTask.BroadcastCnt += 1
			}
		}
		//if goes here, means no consensus reached and timeout, notify chain
		molacp_log.Debugf("<%s> no consensus reached, notify chain", cp.groupId)
	}()

	return nil
}

func (cp *MolassesConsensusProposer) HandleCCReq(msg *quorumpb.ChangeConsensusReqMsg) error {
	molacp_log.Debugf("<%s> HandleCCReq called reqId <%s>, epoch <%d>", cp.groupId, msg.Req.ReqId, msg.Epoch)

	cp.locker.Lock()
	defer cp.locker.Unlock()

	//check if req is from group owner
	if cp.grpItem.OwnerPubKey != msg.Req.SenderPubkey {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is not from group owner, ignore", cp.groupId, msg.Req.ReqId)
		return nil
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
			return nil
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
		return err
	}

	hash := localcrypto.Hash(byts)
	if !bytes.Equal(hash, msg.Req.MsgHash) {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> hash is not same as req.MsgHash, ignore", cp.groupId, msg.Req.ReqId)
		return err
	}

	//verify signature
	verifySign, err := cp.cIface.VerifySign(hash, msg.Req.SenderSign, msg.Req.SenderPubkey)

	if err != nil {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> failed with error <%s>", cp.groupId, msg.Req.ReqId, err.Error())
		return err
	}

	if !verifySign {
		molacp_log.Debug("<%s> HandleCCReq reqid <%s> verify sign failed", cp.groupId, msg.Req.ReqId)
		return err
	}

	if cp.currBftTask != nil {
		if msg.Req.Nonce < cp.currBftTask.Proof.Req.Nonce {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> nonce is not bigger than current nonce, ignore", cp.groupId, msg.Req.ReqId)
			return nil
		}
	}

	//handle new req
	if cp.currBftTask != nil && cp.currBftTask.bftCancelFunc != nil {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> cancel current bft task", cp.groupId, msg.Req.ReqId)
		cp.currBftTask.bftCancelFunc()
		cp.currBftTask = nil
	}

	go func() {
		//check if owner is in the producer list (if I am the owner)
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

		//create resp
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
			return
		}

		hash = localcrypto.Hash(byts)
		ks := nodectx.GetNodeCtx().Keystore
		signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
		resp.MsgHash = hash
		resp.SenderSign = signature

		//create Proof
		proofBundle := &quorumpb.ConsensusProof{
			Epoch: msg.Epoch,
			Req:   msg.Req,
			Resp:  resp,
		}

		//create bft config
		config, err := cp.createBftConfig(msg.Req.ProducerPubkeyList)
		if err != nil {
			molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
			return
		}

		//create new context
		bftCtx, bftCancel := context.WithCancel(cp.chainCtx)

		//create channel to receive bft result
		chBftDone := make(chan *quorumpb.ChangeConsensusResultBundle, 1)

		//create bft task
		cp.currBftTask = &BftTask{
			GOID:          goid(),
			bftCtx:        bftCtx,
			bftCancelFunc: bftCancel,
			bft:           NewPCBft(bftCtx, *config, chBftDone, cp.cIface),
			chBftDone:     chBftDone,
			Proof:         proofBundle,
		}

		cp.currBftTask.bft.AddProof(proofBundle)
		cp.currBftTask.bft.Propose()

		//wait result
		select {
		case <-cp.currBftTask.bftCtx.Done():
			molacp_log.Debugf("<%s> HandleCCReq bft context done, GOID <%d> ", cp.groupId, cp.currBftTask.GOID)
			return
		case result := <-cp.currBftTask.chBftDone:
			molacp_log.Debugf("<%s> HandleCCReq bft done with result", cp.groupId)
			cp.cIface.ChangeConsensusDone(result)

			//cancel current task
			cp.currBftTask.bftCancelFunc()
			//cancel current sender task if any
			if cp.currSendTask != nil && cp.currSendTask.senderCancelFunc != nil {
				cp.currSendTask.senderCancelFunc()
			}

			molacp_log.Debugf("<%s> HandleCCReq bft <%d> done, quit peaceful", cp.groupId, cp.currBftTask.GOID)
			return
		}
	}()

	molacp_log.Debugf("<%s> HandleCCReq done", cp.groupId)
	return nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {

	cp.locker.Lock()
	defer cp.locker.Unlock()

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

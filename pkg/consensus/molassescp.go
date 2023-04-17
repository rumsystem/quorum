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
	Req        *quorumpb.ChangeConsensusReq
	bftCtx     context.Context
	cancelFunc context.CancelFunc
	bft        *PCBft
	chBftDone  chan *quorumpb.ChangeConsensusResultBundle
}

type MolassesConsensusProposer struct {
	grpItem  *quorumpb.GroupItem
	groupId  string
	nodename string
	trxId    string
	cIface   def.ChainMolassesIface
	chainCtx context.Context

	currTask *ConsensusProposeTask
	locker   sync.Mutex

	senderCtx        context.Context
	senderCancelFunc context.CancelFunc

	broadcastCnt int
}

func (cp *MolassesConsensusProposer) NewConsensusProposer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molacp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
	cp.grpItem = item
	cp.groupId = item.GroupId
	cp.nodename = nodename
	cp.trxId = ""
	cp.cIface = iface
	cp.chainCtx = ctx
	cp.broadcastCnt = 0
}

func (cp *MolassesConsensusProposer) StartChangeConsensus(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartChangeConsensus called", cp.groupId)

	cp.locker.Lock()
	defer cp.locker.Unlock()

	//cancel previous sender if any
	if cp.senderCancelFunc != nil {
		cp.senderCancelFunc()
		cp.senderCancelFunc = nil
	}

	//sleep for 1s to make sure sender goroutine is closed
	time.Sleep(1000 * time.Millisecond)

	cp.trxId = trxId

	go func() {
		//TBD get nonce
		nonce := uint64(0)
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

		ks := nodectx.GetNodeCtx().Keystore
		hash := localcrypto.Hash(byts)
		signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
		req.MsgHash = hash
		req.SenderSign = signature

		cp.senderCtx, cp.senderCancelFunc = context.WithCancel(cp.chainCtx)
		cp.broadcastCnt = 0
		for cp.broadcastCnt < int(agrmTickCnt) {
			go func() {
				molacp_log.Debugf("<%s> send req <%s>", cp.groupId, req.ReqId)
				connMgr, err := conn.GetConn().GetConnMgr(cp.groupId)
				if err != nil {
					return
				}
				connMgr.BroadcastPPReq(req)
			}()

			select {
			case <-cp.senderCtx.Done():
				molacp_log.Debugf("<%s> ctx Done, stop sending req ", cp.groupId)
				return
			case <-time.After(time.Duration(req.AgreementTickLenInMs) * time.Millisecond):
				molacp_log.Debugf("<%s> round <%d> timeout", cp.groupId, cp.broadcastCnt)
				cp.broadcastCnt += 1
			}
		}
		//if goes here, means no consensus reached and timeout, notify chain
		molacp_log.Debugf("<%s> no consensus reached, notify chain", cp.groupId)
	}()

	return nil
}

func (cp *MolassesConsensusProposer) HandleCCReq(req *quorumpb.ChangeConsensusReq) error {
	molacp_log.Debugf("<%s> HandleCCReq called reqId <%s>", cp.groupId, req.ReqId)

	cp.locker.Lock()
	defer cp.locker.Unlock()

	//check if req is from group owner
	if cp.grpItem.OwnerPubKey != req.SenderPubkey {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is not from group owner, ignore", cp.groupId, req.ReqId)
		return nil
	}

	//check if I am in the producer list (not owner)
	if !cp.cIface.IsOwner() {
		inTheList := false
		for _, pubkey := range req.ProducerPubkeyList {
			if pubkey == cp.grpItem.UserSignPubkey {
				inTheList = true
				break
			}
		}
		if !inTheList {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is not for me, ignore", cp.groupId, req.ReqId)
			return nil
		}
	}

	//verify if req is valid
	dumpreq := &quorumpb.ChangeConsensusReq{
		ReqId:                req.ReqId,
		GroupId:              req.GroupId,
		Nonce:                req.Nonce,
		ProducerPubkeyList:   req.ProducerPubkeyList,
		AgreementTickLenInMs: req.AgreementTickLenInMs,
		AgreementTickCount:   req.AgreementTickCount,
		StartFromEpoch:       req.StartFromEpoch,
		TrxEpochTickLenInMs:  req.TrxEpochTickLenInMs,
		SenderPubkey:         req.SenderPubkey,
		MsgHash:              nil,
		SenderSign:           nil,
	}

	byts, err := proto.Marshal(dumpreq)
	if err != nil {
		molacp_log.Errorf("<%s> marshal change consensus req failed", cp.groupId)
		return err
	}

	hash := localcrypto.Hash(byts)
	if !bytes.Equal(hash, req.MsgHash) {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> hash is not same as req.MsgHash, ignore", cp.groupId, req.ReqId)
		return fmt.Errorf("req hash is not same as req.MsgHash")
	}

	//verify signature
	verifySign, err := cp.cIface.VerifySign(hash, req.SenderSign, req.SenderPubkey)

	if err != nil {
		molacp_log.Debugf("<%s> HandleCCReq reqid <%s> failed with error <%s>", cp.groupId, req.ReqId, err.Error())
		return err
	}

	if !verifySign {
		return fmt.Errorf("verify signature failed")
	}

	//handle new req
	if cp.currTask != nil {
		molacp_log.Debugf("<%s> StartChangeConsensus, cancel previous task", cp.groupId)
		cp.currTask.cancelFunc()
	}

	//sleep 1s to make sure previous task is closed
	time.Sleep(1000 * time.Millisecond)

	//check if owner is in the producer list
	if cp.cIface.IsOwner() {
		isInProducerList := false
		for _, producer := range req.ProducerPubkeyList {
			if producer == cp.grpItem.UserSignPubkey {
				isInProducerList = true
				break
			}
		}

		//if not add owner to the list to finish consensus
		if !isInProducerList {
			req.ProducerPubkeyList = append(req.ProducerPubkeyList, cp.grpItem.OwnerPubKey)
		}
	}

	go func() {
		//create resp
		resp := &quorumpb.ChangeConsensusResp{
			RespId:       guuid.New().String(),
			GroupId:      req.GroupId,
			SenderPubkey: cp.grpItem.UserSignPubkey,
			Req:          req,
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
		req.MsgHash = hash
		req.SenderSign = signature

		//create Proof
		proofBundle := &quorumpb.ConsensusProof{
			Req:  req,
			Resp: resp,
		}

		//create bft config
		config, err := cp.createBftConfig(req.ProducerPubkeyList)
		if err != nil {
			molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
			return
		}

		//create new context
		taskCtx, taskCancel := context.WithCancel(cp.chainCtx)

		//create channel to receive bft result
		chBftDone := make(chan *quorumpb.ChangeConsensusResultBundle, 1)

		//create bft
		molacp_log.Debugf("<%s> create new bft", cp.groupId)
		bft := NewPCBft(taskCtx, cp.groupId, cp.nodename, *config, chBftDone, cp.cIface)

		task := &ConsensusProposeTask{
			Req:        req,
			bftCtx:     taskCtx,
			cancelFunc: taskCancel,
			bft:        bft,
			chBftDone:  chBftDone,
		}

		cp.currTask = task

		cp.currTask.bft.AddProof(proofBundle)
		cp.currTask.bft.Propose()

		//wait result
		select {
		case <-cp.currTask.bftCtx.Done():
			molacp_log.Debugf("<%s> HandleCCReq bft context done", cp.groupId)
			return
		case result := <-cp.currTask.chBftDone:
			molacp_log.Debugf("<%s> HandleCCReq bft done with result", cp.groupId)

			//cancel current task
			cp.currTask.cancelFunc()

			//cancel current sender task if any
			if cp.senderCancelFunc != nil {
				cp.senderCancelFunc()
				cp.senderCancelFunc = nil
			}

			cp.cIface.ChangeConsensusDone(cp.trxId, result)

			//notify chain
			return
		}
	}()

	molacp_log.Debugf("<%s> HandleCCReq done", cp.groupId)
	return nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)
	if cp.currTask != nil {
		cp.currTask.bft.HandleHBMsg(hbmsg)
	}
	return nil
}

/*
func (cp *MolassesConsensusProposer) HandleBFTTimeout(epoch uint64, reqId string, responsedProducers []string) {
	molacp_log.Debugf("<%s> HandleBFTTimeout called", cp.groupId)

	bundle := &quorumpb.ChangeConsensusResultBundle{
		Req:                cp.CurrReq,
		Resps:              nil,
		Result:             quorumpb.ChangeConsensusResult_TIMEOUT,
		Epoch:              epoch,
		ResponsedProducers: responsedProducers,
	}

	cp.cIface.ChangeConsensusDone(cp.trxId, cp.CurrReq.ReqId, bundle)
}
*/

func (cp *MolassesConsensusProposer) createBftConfig(producers []string) (*Config, error) {
	molacp_log.Debugf("<%s> createBftConfig called", cp.groupId)
	n := len(producers)
	f := 0 // all participant producers(owner included) should agree with the consensus request

	molaproducer_log.Debugf("failable producers <%d>", f)
	batchSize := 1

	config := &Config{
		N:         n,
		f:         f,
		Nodes:     producers,
		BatchSize: batchSize,
		MyPubkey:  cp.grpItem.UserSignPubkey,
	}

	return config, nil
}

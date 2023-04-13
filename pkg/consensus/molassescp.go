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

const DEFAULT_CC_REQ_SEND_INTEVL = 1 * 1000 //1s

type CCRResult uint

const (
	CCRSuccess CCRResult = iota
	CCRFail
	CCRTimeout
)

type MolassesConsensusProposer struct {
	grpItem         *quorumpb.GroupItem
	groupId         string
	nodename        string
	producerspubkey []string

	trxId   string
	CurrReq *quorumpb.ChangeConsensusReq

	cIface        def.ChainMolassesIface
	chainCtx      context.Context
	currCtx       context.Context
	ctxCancelFunc context.CancelFunc

	bft       *PCBft
	chBftDone chan *BftResult

	locker sync.Mutex
}

func (cp *MolassesConsensusProposer) NewConsensusProposer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molacp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
	cp.grpItem = item
	cp.groupId = item.GroupId
	cp.nodename = nodename

	cp.trxId = ""
	cp.CurrReq = nil

	cp.cIface = iface
	cp.chainCtx = ctx
	cp.ctxCancelFunc = nil

	cp.bft = nil
	cp.chBftDone = make(chan *BftResult)
}

func (cp *MolassesConsensusProposer) StartChangeConsensus(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartChangeConsensus called", cp.groupId)

	cp.locker.Lock()
	defer cp.locker.Unlock()

	//stop all previous underlay go routine by call ctx cancel
	if cp.ctxCancelFunc != nil {
		cp.ctxCancelFunc()
	}

	//sleep for 1s to make sure all go routine stopped???
	time.Sleep(1000 * time.Millisecond)

	cp.trxId = trxId
	cp.producerspubkey = producers

	//create new ctx with cancel
	cp.currCtx, cp.ctxCancelFunc = context.WithCancel(cp.chainCtx)

	//tbd get current nonce
	nonce := uint64(0)

	//create req
	req := &quorumpb.ChangeConsensusReq{
		ReqId:                guuid.New().String(),
		GroupId:              cp.groupId,
		Nonce:                nonce,
		ProducerPubkeyList:   cp.producerspubkey,
		AgreementTickLenInMs: agrmTickLen,
		AgreementTickCount:   agrmTickCnt,
		StartFromEpoch:       fromNewEpoch,
		TrxEpochTickLenInMs:  trxEpochTickLen,
		SenderPubkey:         cp.grpItem.UserSignPubkey,
	}

	byts, err := proto.Marshal(req)
	if err != nil {
		molacp_log.Errorf("<%s> marshal change consensus req failed", cp.groupId)
		return err
	}

	ks := nodectx.GetNodeCtx().Keystore

	hash := localcrypto.Hash(byts)
	signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
	req.MsgHash = hash
	req.SenderSign = signature

	go func() {
		molacp_log.Debugf("<%s> create ticker to send req<%s>", cp.groupId, req.ReqId)
		ticker := time.NewTicker(time.Duration(DEFAULT_CC_REQ_SEND_INTEVL) * time.Millisecond)
		for {
			select {
			case <-cp.currCtx.Done():
				molacp_log.Debugf("<%s> ctx Done, stop sending req", cp.groupId)
				return
			case <-ticker.C:
				molacp_log.Debugf("<%s> send req <%s>", cp.groupId, req.ReqId)
				connMgr, err := conn.GetConn().GetConnMgr(cp.groupId)
				if err != nil {
					return
				}
				connMgr.BroadcastPPReq(req)
			}
		}
	}()

	molacp_log.Debugf("here???")

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

	if cp.CurrReq != nil {
		if cp.CurrReq.ReqId == req.ReqId {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is same as current reqid, ignore", cp.groupId, req.ReqId)
			return nil
		} else if cp.CurrReq.Nonce > req.Nonce {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> nonce <%d> is smaller than current reqid <%s> nonce <%d>, ignore", cp.groupId, req.ReqId, req.Nonce, cp.CurrReq.ReqId, cp.CurrReq.Nonce)
			return nil
		}
	}

	cp.CurrReq = req

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

	//check if owner is in the producer list
	if cp.cIface.IsOwner() {
		isInProducerList := false
		for _, producer := range req.ProducerPubkeyList {
			if producer == cp.grpItem.UserSignPubkey {
				isInProducerList = true
				break
			}
		}

		if !isInProducerList {
			req.ProducerPubkeyList = append(req.ProducerPubkeyList, cp.grpItem.OwnerPubKey)
		}
	}

	//create
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
		return err
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
	config, err := cp.createBftConfig()
	if err != nil {
		molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
		return err
	}

	//create and start bft
	molacp_log.Debugf("<%s> create bft", cp.groupId)
	cp.bft = NewPCBft(cp.currCtx, cp.groupId, cp.nodename, *config, req.AgreementTickLenInMs, req.AgreementTickCount, cp.chBftDone)
	cp.bft.AddProof(proofBundle)
	cp.bft.Propose()

	for {
		select {
		case <-cp.currCtx.Done():
			molacp_log.Debugf("<%s> HandleCCReq local context done", cp.groupId)
			return nil
		case <-cp.chBftDone:
			molacp_log.Debugf("<%s> HandleCCReq bft done", cp.groupId)
			//handle resutl
			//notify chain
			//finish all local goroutine
			cp.ctxCancelFunc()
			molacp_log.Debugf("<%s> HandleCCReq reqId <%s> done", cp.groupId, req.ReqId)
			return nil
		}
	}
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)
	if cp.bft != nil {
		cp.bft.HandleHBMsg(hbmsg)
	}
	return nil
}

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

func (cp *MolassesConsensusProposer) createBftConfig() (*Config, error) {
	molacp_log.Debugf("<%s> createBftConfig called", cp.groupId)

	var producerNodes []string

	n := len(producerNodes)
	f := 0 // all participant producers(owner included) should agree with the consensus request

	molaproducer_log.Debugf("failable producers <%d>", f)
	batchSize := 1

	config := &Config{
		N:         n,
		f:         f,
		Nodes:     producerNodes,
		BatchSize: batchSize,
		MyPubkey:  cp.grpItem.UserSignPubkey,
	}

	return config, nil
}

/*

func (cp *MolassesConsensusProposer) verifyRawResult(rawResult map[string][]byte) (bool, map[string]*quorumpb.ConsensusProof) {
	pcbft_log.Debugf("<%s> verifyRawResult called", cp.groupId)

	//convert rawResultMap to proof map
	proofMap := make(map[string]*quorumpb.ConsensusProof)
	for k, v := range rawResult {
		proof := &quorumpb.ConsensusProof{}
		err := proto.Unmarshal(v, proof)
		if err != nil {
			pcbft_log.Errorf("<%s> unmarshal consensus proof failed", cp.groupId)
			return false, proofMap
		}
		proofMap[k] = proof
	}

	//check if the maps contains all key from producerspubkey
	for _, pubkey := range Config.ProducersPubkey {
		proof, ok := proofMap[pubkey]
		if !ok {
			pcbft_log.Errorf("<%s> proof map does not contains producerPubkey <%s>", cp.groupId, pubkey)
			return false, proofMap
		}

		//check if sender is as same as producerPubkey
		if proof.Resp.SenderPubkey != pubkey {
			pcbft_log.Errorf("<%s> proof sender is not same as producerPubkey", cp.groupId)
			return false, proofMap
		}

		//check if all req are the same as original req
		if !proto.Equal(proof.Req, cp.CurrReq) {
			pcbft_log.Errorf("<%s> proof req is not same as original req", cp.groupId)
			return false, proofMap
		}

		//check if all resp sign against the same original req
		if !proto.Equal(proof.Req, proof.Resp.Req) {
			pcbft_log.Errorf("<%s> proof req is not same as proof resp req", cp.groupId)
			return false, proofMap
		}

		//check if all resp sign is valid
		dumpResp := &quorumpb.ChangeConsensusResp{
			RespId:       proof.Resp.RespId,
			GroupId:      proof.Resp.GroupId,
			SenderPubkey: proof.Resp.SenderPubkey,
			Req:          proof.Resp.Req,
			MsgHash:      nil,
			SenderSign:   nil,
		}

		byts, err := proto.Marshal(dumpResp)
		if err != nil {
			pcbft_log.Errorf("<%s> marshal change consensus resp failed", cp.groupId)
			return false, proofMap
		}

		hash := localcrypto.Hash(byts)
		if !bytes.Equal(hash, proof.Resp.MsgHash) {
			pcbft_log.Errorf("<%s> proof resp hash is not same as original hash", cp.groupId)
			return false, proofMap
		}

		isValid, err := cp.cIface.VerifySign(hash, proof.Resp.SenderSign, proof.Resp.SenderPubkey)
		if err != nil {
			pcbft_log.Errorf("<%s> verify proof resp sign failed", cp.groupId)
			return false, proofMap
		}

		if !isValid {
			pcbft_log.Errorf("<%s> proof resp sign is not valid", cp.groupId)
			return false, proofMap
		}
	}

	return true, proofMap
}
*/

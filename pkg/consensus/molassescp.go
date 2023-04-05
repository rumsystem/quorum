package consensus

import (
	"bytes"
	"fmt"
	"sync"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var molacp_log = logging.Logger("cp")

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
	cIface          def.ChainMolassesIface
	producerspubkey []string
	trxId           string
	bft             *PCBft
	CurrReq         *quorumpb.ChangeConsensusReq
	ReqSender       *CCReqSender
	locker          sync.Mutex
}

func (cp *MolassesConsensusProposer) NewConsensusProposer(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molacp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
	cp.grpItem = item
	cp.cIface = iface
	cp.nodename = nodename
	cp.groupId = item.GroupId
	cp.trxId = ""
	cp.bft = nil
	cp.CurrReq = nil
	cp.ReqSender = nil
}

func (cp *MolassesConsensusProposer) StartChangeConsensus(producerList *quorumpb.BFTProducerBundleItem, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartChangeConsensus called", cp.groupId)

	cp.locker.Lock()
	defer cp.locker.Unlock()

	//stop current bft
	if cp.bft != nil {
		molacp_log.Debugf("<%s> Stop current bft", cp.groupId)
		cp.bft.Stop()
	}

	if cp.ReqSender != nil {
		molacp_log.Debugf("<%s> Stop current ReqSender", cp.groupId)
		cp.ReqSender.StopSending()
	}

	if cp.CurrReq != nil {
		molacp_log.Debugf("<%s> clear CurrReq", cp.groupId)
		cp.CurrReq = nil
	}

	cp.trxId = trxId

	//tbd get current nonce
	nonce := uint64(0)

	var pubkeys []string
	for _, producer := range producerList.Producers {
		pubkeys = append(pubkeys, producer.ProducerPubkey)
	}
	cp.producerspubkey = append(cp.producerspubkey, pubkeys...)

	req := &quorumpb.ChangeConsensusReq{
		ReqId:                guuid.New().String(),
		GroupId:              cp.groupId,
		Nonce:                nonce,
		ProducerPubkeyList:   pubkeys,
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
	hashResult := localcrypto.Hash(hash)
	signature, _ := ks.EthSignByKeyName(cp.groupId, hashResult)
	req.MsgHash = hash
	req.SenderSign = signature

	//create req sender and start send req
	sender := NewCCReqSender(cp.groupId)
	cp.ReqSender.SendCCReq(req)
	cp.ReqSender = sender

	return nil
}

func (cp *MolassesConsensusProposer) HandleCCReq(req *quorumpb.ChangeConsensusReq) error {
	molacp_log.Debugf("<%s> HandleCCReq called", cp.groupId)

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
		return err
	}
	if !verifySign {
		return fmt.Errorf("verify signature failed")
	}

	//stop current bft
	if cp.bft != nil {
		cp.bft.Stop()
	}

	//stop current sender
	if cp.ReqSender != nil {
		cp.ReqSender.StopSending()
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

	//create bft config
	config, err := cp.createBftConfig()
	if err != nil {
		molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
		return err
	}

	//create Proof
	proofBundle := &quorumpb.ConsensusProof{
		Req:  req,
		Resp: resp,
	}

	//create and start bft
	molacp_log.Debugf("<%s> create bft", cp.groupId)
	cp.bft = NewPCBft(*config, cp, req.AgreementTickLenInMs, req.AgreementTickCount)
	cp.bft.Start()
	cp.bft.AddProof(proofBundle)

	return nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)

	if cp.bft != nil {
		cp.bft.HandleHBMsg(hbmsg)
	}
	return nil
}

func (cp *MolassesConsensusProposer) HandleBftDone(epoch uint64, rawResult map[string][]byte) {
	molacp_log.Debugf("<%s> HandleBftDone called", cp.groupId)

	isOk, proofs := cp.verifyRawResult(rawResult)

	bundle := &quorumpb.ChangeConsensusResultBundle{}

	//dump req to bundle
	bundle.Req = proofs[cp.grpItem.OwnerPubKey].Req

	//dump all resps to bundle
	for _, proof := range proofs {
		bundle.Resps = append(bundle.Resps, proof.Resp)
	}

	//dump epoch to bundle
	bundle.Epoch = epoch

	//dump all producerpubkeys to bundle
	bundle.ResponsedProducers = cp.producerspubkey

	if !isOk {
		molacp_log.Errorf("<%s> verify raw result failed, bft finished but agreement not made", cp.groupId)
		bundle.Result = quorumpb.ChangeConsensusResult_FAIL
	} else {
		bundle.Result = quorumpb.ChangeConsensusResult_SUCCESS
	}

	//notify chain
	cp.cIface.ChangeConsensusDone(cp.trxId, cp.CurrReq.ReqId, bundle)
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

func (cp *MolassesConsensusProposer) verifyRawResult(rawResult map[string][]byte) (bool, map[string]*quorumpb.ConsensusProof) {
	molacp_log.Debugf("<%s> verifyRawResult called", cp.groupId)

	//convert rawResultMap to proof map
	proofMap := make(map[string]*quorumpb.ConsensusProof)
	for k, v := range rawResult {
		proof := &quorumpb.ConsensusProof{}
		err := proto.Unmarshal(v, proof)
		if err != nil {
			molacp_log.Errorf("<%s> unmarshal consensus proof failed", cp.groupId)
			return false, proofMap
		}
		proofMap[k] = proof
	}

	//check if the maps contains all key from producerspubkey
	for _, pubkey := range cp.producerspubkey {
		proof, ok := proofMap[pubkey]
		if !ok {
			molacp_log.Errorf("<%s> proof map does not contains producerPubkey <%s>", cp.groupId, pubkey)
			return false, proofMap
		}

		//check if sender is as same as producerPubkey
		if proof.Resp.SenderPubkey != pubkey {
			molacp_log.Errorf("<%s> proof sender is not same as producerPubkey", cp.groupId)
			return false, proofMap
		}

		//check if all req are the same as original req
		if !proto.Equal(proof.Req, cp.CurrReq) {
			molacp_log.Errorf("<%s> proof req is not same as original req", cp.groupId)
			return false, proofMap
		}

		//check if all resp sign against the same original req
		if !proto.Equal(proof.Req, proof.Resp.Req) {
			molacp_log.Errorf("<%s> proof req is not same as proof resp req", cp.groupId)
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
			molacp_log.Errorf("<%s> marshal change consensus resp failed", cp.groupId)
			return false, proofMap
		}

		hash := localcrypto.Hash(byts)
		if !bytes.Equal(hash, proof.Resp.MsgHash) {
			molacp_log.Errorf("<%s> proof resp hash is not same as original hash", cp.groupId)
			return false, proofMap
		}

		isValid, err := cp.cIface.VerifySign(hash, proof.Resp.SenderSign, proof.Resp.SenderPubkey)
		if err != nil {
			molacp_log.Errorf("<%s> verify proof resp sign failed", cp.groupId)
			return false, proofMap
		}

		if !isValid {
			molacp_log.Errorf("<%s> proof resp sign is not valid", cp.groupId)
			return false, proofMap
		}
	}

	return true, proofMap
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

package consensus

import (
	"bytes"
	"fmt"

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

func (cp *MolassesConsensusProposer) StartNewCCRBoradcast(producerList *quorumpb.BFTProducerBundleItem, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> StartNewCCRBoradcast called", cp.groupId)

	//stop current bft
	if cp.bft != nil {
		molacp_log.Debugf("<%s> Stop current bft", cp.groupId)
		cp.bft.Stop()
	}

	cp.trxId = trxId

	//get nonce

	var pubkeys []string
	for _, producer := range producerList.Producers {
		pubkeys = append(pubkeys, producer.ProducerPubkey)
	}

	cp.producerspubkey = append(cp.producerspubkey, pubkeys...)

	req := &quorumpb.ChangeConsensusReq{
		ReqId:                guuid.New().String(),
		GroupId:              cp.groupId,
		Nonce:                0,
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

	cp.CurrReq = req

	//create req sender and send req
	sender := NewCCReqSender(cp.groupId, cp.CurrReq.AgreementTickLenInMs, cp.CurrReq.AgreementTickCount, cp)
	cp.ReqSender = sender
	cp.ReqSender.SendCCReq(cp.CurrReq)

	return nil
}

func (cp *MolassesConsensusProposer) HandleCCReq(req *quorumpb.ChangeConsensusReq) error {
	molacp_log.Debugf("<%s> HandleCCReq called", cp.groupId)
	if cp.CurrReq != nil {
		if cp.CurrReq.ReqId == req.ReqId {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is same as current reqid, ignore", cp.groupId, req.ReqId)
		} else if cp.CurrReq.Nonce > req.Nonce {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> nonce <%d> is smaller than current reqid <%s> nonce <%d>, ignore", cp.groupId, req.ReqId, req.Nonce, cp.CurrReq.ReqId, cp.CurrReq.Nonce)
		}
	} else {
		//check if req is from group owner
		if cp.grpItem.OwnerPubKey != req.SenderPubkey {
			molacp_log.Debugf("<%s> HandleCCReq reqid <%s> is not from group owner, ignore", cp.groupId, req.ReqId)
			return nil
		}

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

		//
		cp.CurrReq = req
		//add pubkeys for all producers
		cp.producerspubkey = append(cp.producerspubkey, req.ProducerPubkeyList...)

		//create resp
		bundle := &quorumpb.ConsensusBundle{
			Req: req,
		}

		shouldCreateResp := true
		if cp.cIface.IsOwner() {
			//check if I am in the producer list
			isInProducerList := false
			for _, producer := range req.ProducerPubkeyList {
				if producer == cp.grpItem.UserSignPubkey {
					isInProducerList = true
					break
				}
			}

			if !isInProducerList {
				shouldCreateResp = false
			}
		}

		//create resp
		if shouldCreateResp {
			resp := &quorumpb.ChangeConsensusResp{
				RespId:       guuid.New().String(),
				GroupId:      req.GroupId,
				SenderPubkey: cp.grpItem.UserSignPubkey,
				Req:          req,
				MsgHash:      nil,
				SenderSign:   nil,
			}

			byts, err := proto.Marshal(dumpreq)
			if err != nil {
				molacp_log.Errorf("<%s> marshal change consensus resp failed", cp.groupId)
				return err
			}

			hash := localcrypto.Hash(byts)
			ks := nodectx.GetNodeCtx().Keystore
			signature, _ := ks.EthSignByKeyName(cp.groupId, hash)
			req.MsgHash = hash
			req.SenderSign = signature
			bundle.Resp = resp
		}

		//create bft config
		config, err := cp.createBftConfig()
		if err != nil {
			molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
			return err
		}

		//create bft
		molacp_log.Debugf("<%s> create bft", cp.groupId)
		cp.bft = NewPCBft(*config, cp)
		cp.bft.Start()

		cp.bft.AddBundle(bundle)
	}

	return nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)

	if cp.bft != nil {
		cp.bft.HandleHBMsg(hbmsg)
	}
	return nil
}

func (cp *MolassesConsensusProposer) HandleCCRResult(reqid string, result CCRResult) {
	if reqid != cp.CurrReq.ReqId {
		molacp_log.Debugf("<%s> HandleCCRResult reqid <%s> is not same as current reqid <%s>, ignore", cp.groupId, reqid, cp.CurrReq.ReqId)
		return
	}

	switch result {
	case CCRSuccess:
		molacp_log.Debugf("<%s> HandleCCRResult CCRSuccess", cp.groupId)
	case CCRFail:
		molacp_log.Debugf("<%s> HandleCCRResult CCRFail", cp.groupId)
	case CCRTimeout:
		molacp_log.Debugf("<%s> HandleCCRResult CCRTimeout", cp.groupId)
	default:
	}
}

func (cp *MolassesConsensusProposer) createBftConfig() (*Config, error) {
	molacp_log.Debugf("<%s> createBftConfig called", cp.groupId)

	var producerNodes []string

	n := len(producerNodes)
	f := 0 // all participant producers should agree with the consensus request

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

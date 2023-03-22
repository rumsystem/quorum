package consensus

import (
	"bytes"
	"encoding/hex"
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

func (cp *MolassesConsensusProposer) AddNewConsensusItem(producerList *quorumpb.BFTProducerBundleItem, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> AddNewConsensusItem called", cp.groupId)

	//stop current bft
	if cp.bft != nil {
		cp.bft.Stop()
	}

	cp.trxId = trxId

	//create ChangeConsensusReq
	//load current group consensus proposer nonce
	nouce, err := nodectx.GetNodeCtx().GetChainStorage().GetConsensusProposeNonce(cp.groupId)
	if err != nil {
		molacp_log.Errorf("<%s> GetConsensusProposeNonce failed", cp.groupId)
		return err
	}

	var pubkeys []string
	for _, producer := range producerList.Producers {
		pubkeys = append(pubkeys, producer.ProducerPubkey)
	}

	req := &quorumpb.ChangeConsensusReq{
		ReqId:                guuid.New().String(),
		GroupId:              cp.groupId,
		Nonce:                nouce + 1,
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
	encodedSign := hex.EncodeToString(signature)

	req.MsgHash = hash
	req.SenderPubkey = encodedSign

	cp.CurrReq = req

	//add pubkeys for all producers
	for _, producer := range producerList.Producers {
		cp.producerspubkey = append(cp.producerspubkey, producer.ProducerPubkey)
	}

	//create bft config
	config, err := cp.createBftConfig()
	if err != nil {
		molacp_log.Errorf("<%s> create bft config failed", cp.groupId)
		return err
	}

	//create bft
	cp.bft = NewPCBft(*config, cp)

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

		cp.createBftConfig()

		//verify req
		//copy req	by value
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

func (cp *MolassesConsensusProposer) createBftConfig() (*Config, error) {
	molacp_log.Debugf("<%s> createBftConfig called", cp.groupId)

	var producerNodes []string
	shouldAddOwner := true

	for _, pubkey := range cp.producerspubkey {
		if pubkey == cp.grpItem.OwnerPubKey {
			shouldAddOwner = false
		}
		molaproducer_log.Debugf(">>> add producer pubkey <%s>", pubkey)
	}

	if shouldAddOwner {
		producerNodes = append(producerNodes, cp.grpItem.OwnerPubKey)
	}

	n := len(producerNodes)
	f := (n - 1) / 3

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

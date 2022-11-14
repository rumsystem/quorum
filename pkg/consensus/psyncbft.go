package consensus

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pbft_log = logging.Logger("pbft")

type PSyncBft struct {
	Config
	PSyncer  *MolassesPSync
	acsInsts map[string]*PSyncACS //map key is sessionId
}

func NewPSyncBft(cfg Config, psyncer *MolassesPSync) *PSyncBft {
	pbft_log.Debugf("NewPSyncBft called")
	return &PSyncBft{
		Config:   cfg,
		PSyncer:  psyncer,
		acsInsts: make(map[string]*PSyncACS),
	}
}

func (pbft *PSyncBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	pbft_log.Debugf("HandleMessage called,  <%s>", hbmsg.MsgId)

	acs, ok := pbft.acsInsts[hbmsg.SessionId]

	if !ok {
		acs = NewPSyncACS(pbft.Config, pbft, hbmsg.SessionId)
		pbft.acsInsts[hbmsg.SessionId] = acs
		pbft_log.Debugf("Create new ACS %d", hbmsg.SessionId)
	}

	return acs.HandleMessage(hbmsg)
}

func (pbft *PSyncBft) AcsDone(sessionId string, result map[string][]byte) {
	pbft_log.Debugf("AcsDone called, SessionId <%s>", sessionId)

	//make a consensusResp and send it out

	//get producer registered trx
	trx, err := nodectx.GetNodeCtx().GetChainStorage().GetUpdProducerListTrx(pbft.PSyncer.groupId, pbft.PSyncer.nodename)
	if err != nil && err.Error() != "Key not found" {
		pbft_log.Debugf(err.Error())
		return
	}

	//get current producers
	prds := &quorumpb.PSyncProducerItem{}
	prds.Producers = append(prds.Producers, pbft.Config.Nodes...)

	//TBD fill withness
	witness := []*quorumpb.Witnesses{}

	resp := &quorumpb.ConsensusResp{
		CurChainEpoch: pbft.PSyncer.grpItem.Epoch,
		CurProducer:   prds,
		Witesses:      witness,
		ProducerProof: trx,
	}

	payload, err := proto.Marshal(resp)
	if err != nil {
		pbft_log.Debugf(err.Error())
		return
	}

	consusResp := &quorumpb.ConsensusMsg{
		GroupId:      pbft.PSyncer.groupId,
		SessionId:    sessionId,
		MsgType:      quorumpb.ConsensusType_RESP,
		Payload:      payload,
		SenderPubkey: pbft.MySignPubkey,
		TimeStamp:    time.Now().UnixNano(),
	}

	bbytes, err := proto.Marshal(consusResp)
	if err != nil {
		pbft_log.Debugf(err.Error())
		return
	}

	msgHash := localcrypto.Hash(bbytes)

	var signature []byte
	ks := localcrypto.GetKeystore()
	signature, err = ks.EthSignByKeyName(pbft.PSyncer.groupId, msgHash, pbft.PSyncer.nodename)

	if err != nil {
		pbft_log.Debugf(err.Error())
		return
	}

	if len(signature) == 0 {
		pbft_log.Debugf("create signature failed")
		return
	}

	//save hash and signature
	consusResp.MsgHash = msgHash
	consusResp.SenderSign = signature

	//send consensusResp out
	connMgr, err := conn.GetConn().GetConnMgr(pbft.PSyncer.groupId)
	if err != nil {
		pbft_log.Debugf(err.Error())
		return
	}

	err = connMgr.SentConsensusMsgPubsub(consusResp, conn.ProducerChannel)
	if err != nil {
		pbft_log.Debugf(err.Error())
		return
	}

	//clear acs
	pbft.acsInsts[sessionId] = nil
	delete(pbft.acsInsts, sessionId)

	pbft_log.Debugf("Remove acs %s", sessionId)
}

func (pbft *PSyncBft) AddConsensusReq(req *quorumpb.ConsensusMsg) error {
	pbft_log.Debug("Propose called")

	datab, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	_, ok := pbft.acsInsts[req.SessionId]
	if !ok {
		acs := NewPSyncACS(pbft.Config, pbft, req.SessionId)
		pbft.acsInsts[req.SessionId] = acs
	}

	return pbft.acsInsts[req.SessionId].InputValue(datab)
}

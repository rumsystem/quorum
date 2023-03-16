package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var ppbft_log = logging.Logger("ppbft")

var DEFAULT_PRODUCER_PROPOSE_PULSE = 1 * 1000 //1s

type PPTask struct {
	Epoch       uint64
	ProposeData []byte
	acsInsts    *PPAcs
}

type PPBft struct {
	Config
	groupId  string
	pp       *MolassesProducerProposer
	currTask *PPTask
}

func NewPPBft(cfg Config, pp *MolassesProducerProposer) *PPBft {
	ppbft_log.Debugf("NewPPBft called")
	return &PPBft{
		Config:   cfg,
		groupId:  pp.groupId,
		pp:       pp,
		currTask: nil,
	}
}

func (bft *PPBft) HandleHBMessage(hbmsg *quorumpb.HBMsgv1) error {
	ppbft_log.Debugf("<%s> HandleHBMessage called, Epoch <%d>", bft.groupId, hbmsg.Epoch)
	return nil
}

func (bft *PPBft) Start() error {
	return nil
}

func (bft *PPBft) Stop() error {
	return nil
}

func (ppbft *PPBft) AcsDone(epoch uint64, result map[string][]byte) {
	ppbft_log.Debugf("AcsDone called, epoch <%d>", epoch)

	/*

		//get producer registered trx
		trx, err := nodectx.GetNodeCtx().GetChainStorage().GetUpdProducerListTrx(pbft.PSyncer.groupId, pbft.PSyncer.nodename)
		if err != nil && err.Error() != "Key not found" {
			pbft_log.Debugf(err.Error())
			return
		}

		//get current producers
		prds := &quorumpb.PSyncProducerItem{}
		prds.Producers = append(prds.Producers, pbft.Config.Nodes...)

		resp := &quorumpb.PSyncResp{
			GroupId:           pbft.PSyncer.groupId,
			SessionId:         sessionId,
			SenderPubkey:      pbft.MyPubkey,
			MyCurEpoch:        pbft.PSyncer.cIface.GetCurrEpoch(),
			MyCurProducerList: prds,
			ProducerProof:     trx,
		}

		bbytes, err := proto.Marshal(resp)
		if err != nil {
			pbft_log.Debugf(err.Error())
			return
		}

		//sign it
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

		resp.SenderSign = signature
		pbft_log.Debugf("PSyncResp for Session <%s> created", sessionId)

		//send consensusResp out
		connMgr, err := conn.GetConn().GetConnMgr(pbft.PSyncer.groupId)
		if err != nil {
			pbft_log.Debugf(err.Error())
			return
		}

		payload, err := proto.Marshal(resp)
		if err != nil {
			pbft_log.Debugf(err.Error())
			return
		}

		pmsg := &quorumpb.PSyncMsg{
			MsgType: quorumpb.PSyncMsgType_PSYNC_RESP,
			Payload: payload,
		}

		pbft_log.Debugf("Send ConsensusResp for Session <%s>", sessionId)
		err = connMgr.BroadcastPSyncMsg(pmsg)
		if err != nil {
			pbft_log.Debugf(err.Error())
			return
		}

		pbft_log.Debugf("Resp for Session <%s> done, delete and clear ACS", sessionId)
		//clear acs
		pbft.acsInsts[sessionId] = nil
		delete(pbft.acsInsts, sessionId)
	*/
}

func (ppbft *PPBft) AddProducerProposal(req *quorumpb.ProducerProposalReq) error {
	ppbft_log.Debugf("AddProducerProposal called, SessionId <%s> ", req.ReqId)

	/*
		datab, err := proto.Marshal(req)
		if err != nil {
			return err
		}

		_, ok := pbft.acsInsts[req.SessionId]
		if !ok {
			pbft_log.Debugf("Create new ACS with sessionId <%s>", req.SessionId)
			acs := NewPSyncACS(pbft.Config, pbft, req.SessionId)
			pbft.acsInsts[req.SessionId] = acs
		}

		return pbft.acsInsts[req.SessionId].InputValue(datab)
	*/

	return nil
}

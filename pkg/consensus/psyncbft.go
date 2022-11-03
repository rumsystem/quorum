package consensus

import (
	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
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
	/*


		//get producer registered trx
		trx, err := nodectx.GetNodeCtx().GetChainStorage().GetUpdProducerListTrx(pbft.PSyncer.groupId, pbft.PSyncer.nodename)
		if err != nil && err.Error() != "Key not found" {
			pbft_log.Debugf(err.Error())
			return err
		}

		pSyncMsg := &quorumpb.PSyncMsg{
			SessionId:    uuid.NewString(),
			CurrentEpoch: pbft.PSyncer.grpItem.Epoch,
			NodeStatus:   quorumpb.PSyncNodeStatus_NODE_SYNCING,
			ProofTrx:     trx,
			PSyncItems:   []*quorumpb.PSyncItem{}, //TBD should fill more item if needed
			TimeStamp:    time.Now().UnixNano(),
			Memo:         "",
			SenderPubkey: pbft.MySignPubkey,
		}

		bbytes, err := proto.Marshal(pSyncMsg)
		if err != nil {
			return err
		}

		msgHash := localcrypto.Hash(bbytes)

		var signature []byte
		ks := localcrypto.GetKeystore()
		signature, err = ks.EthSignByKeyName(pbft.PSyncer.groupId, msgHash, pbft.PSyncer.nodename)

		if err != nil {
			return err
		}

		if len(signature) == 0 {
			return errors.New("create signature failed")
		}

		//save hash and signature
		pSyncMsg.PSyncMsgHash = msgHash
		pSyncMsg.SenderSign = signature

		//try propose to BFT
		msgb, err := proto.Marshal(pSyncMsg)
		if err != nil {
			return err
		}

		var acs *PSyncACS
		acs, ok := pbft.acsInsts[pSyncMsg.SenderPubkey]
		if !ok {
			//create and save psync acs instance
			acs = NewPSyncACS(pbft.Config, pbft, pSyncMsg.SessionId)
			pbft.acsInsts[pSyncMsg.SessionId] = acs
		}

		   isProducer := false
		   pbft_log.Debugf("check who am I")

		   	for _, producerId := range pbft.Config.Nodes {
		   		if pbft.PSyncer.grpItem.UserSignPubkey == producerId {
		   			pbft_log.Debugf("I am producer <%s>", pbft.PSyncer.grpItem.UserSignPubkey)
		   			isProducer = true
		   			break
		   		}
		   	}

		   //get the largest epoch and ignore the epoch less than my current epoch

		   	for key, item := range result {
		   		psyncMsg := &quorumpb.PSyncMsg{}
		   		err := proto.Unmarshal(item, psyncMsg)
		   		if err != nil {
		   			pbft_log.Debug("Something wrong %s", err.Error())
		   		}
		   		pbft_log.Debug("%v", psyncMsg)
		   	}
	*/
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

package consensus

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var pbft_log = logging.Logger("pbft")

type PSyncBft struct {
	Config
	PSyncer  *MolassesPSyncer
	acsInsts map[string]*PSyncACS //map key is sessionId
}

func NewPSyncBft(cfg Config, psyncer *MolassesPSyncer) *PSyncBft {
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

	for key, item := range result {
		pbft_log.Debugf("NodeID <%s>", key)
		psyncMsg := &quorumpb.PSyncMsg{}
		err := proto.Unmarshal(item, psyncMsg)
		if err != nil {
			pbft_log.Debug("Something wrong %s", err.Error())
		}
		pbft_log.Debug("%v", psyncMsg)
	}
}

func (pbft *PSyncBft) Propose() error {
	pbft_log.Debug("AcsDone called, sessionId")

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

	acs.InputValue(msgb)

	return nil
}

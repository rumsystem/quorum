package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pbft_log = logging.Logger("pbft")

type PSyncBft struct {
	Config
	PSyncer  *MolassesPSyncer
	acsInsts map[string]*TrxACS //map key is epoch
}

func NewPSyncBft(cfg Config, psyncer *MolassesPSyncer) *PSyncBft {
	pbft_log.Debugf("NewPSyncBft called")
	return &PSyncBft{
		Config:   cfg,
		PSyncer:  psyncer,
		acsInsts: make(map[string]*TrxACS),
	}
}

func (pbft *PSyncBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	pbft_log.Debugf("HandleMessage called,  <%s>", hbmsg.MsgId)
	/*
		acs, ok := bft.acsInsts[hbmsg.Epoch]

			if !ok {
				if hbmsg.Epoch <= bft.producer.grpItem.Epoch {
					bft_log.Warnf("message from old epoch, ignore")
					return nil
				}
				acs = NewACS(bft.Config, bft, hbmsg.Epoch)
				bft.acsInsts[hbmsg.Epoch] = acs
				bft_log.Debugf("Create new ACS %d", hbmsg.Epoch)
			}

			return acs.HandleMessage(hbmsg)
	*/
	return nil
}

func (pbft *PSyncBft) AcsDone(sessionId string, result map[string][]byte) {
	pbft_log.Debugf("AcsDone called")
}

func (pbft *PSyncBft) propose() error {

	/*
		//get producer registered trx
		//TBD : need add a fake trx for owner just after create the group
		trx, err := nodectx.GetNodeCtx().GetChainStorage().GetUpdProducerListTrx(pbft.PSyncer.groupId, pbft.PSyncer.nodename)
		if err != nil {
			return err
		}

		pSyncMsg := &quorumpb.PSyncMsg{
			SessionId:    uuid.NewString(),
			CurrentEpoch: pbft.PSyncer.grpItem.Epoch,
			NodeStatus:   quorumpb.PSyncNodeStatus_NODE_SYNCING,
			ProofTrx:     trx,
			PSyncItems:   []*quorumpb.PSyncItem{},
			TimeStamp:    0,
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
		_, err := proto.Marshal(pSyncMsg)

		if err != nil {
			return err
		}

		_, ok := pbft.acsInsts[pSyncMsg.SenderPubkey]
		if !ok {
			acs := NewACS(pBft.Config, pBft)
		}

	*/
	return nil
}

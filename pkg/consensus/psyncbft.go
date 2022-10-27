package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pbft_log = logging.Logger("pbft")

type PSyncerBft struct {
	Config
	PSyncer  *MolassesPSyncer
	acsInsts map[string]*ACS //map key is epoch
}

func NewPSyncBft(cfg Config, psyncer *MolassesPSyncer) *PSyncerBft {
	pbft_log.Debugf("NewPSyncBft called")
	return &PSyncerBft{
		Config:   cfg,
		PSyncer:  psyncer,
		acsInsts: make(map[string]*ACS),
	}
}

func (pbft *PSyncerBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	pbft_log.Debugf("HandleMessage called, Epoch <%d>", hbmsg.Epoch)

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

func (pbft *PSyncerBft) AcsDone(string, result map[string][]byte) {
	pbft_log.Debugf("AcsDone called")
}

func (pbft *PSyncerBft) propose(epoch int64) error {
	/*
		trxs, err := hb.txBuffer.GetNRandTrx(hb.BatchSize)
		if err != nil {
			return err
		}

		//nothing to propose
		if len(trxs) == 0 {
			acs_log.Infof("trx queue empty, nothing to propose")
			return nil
		}

		trxBundle := &quorumpb.HBTrxBundle{}
		for _, trx := range trxs {
			trxBundle.Trxs = append(trxBundle.Trxs, trx)
		}

		datab, err := proto.Marshal(trxBundle)
		acs, ok := hb.acsInsts[epoch]
		if !ok {
			acs = NewACS(hb.Config, hb, epoch)
			hb.acsInsts[epoch] = acs
		}

		return hb.acsInsts[epoch].InputValue(datab)
	*/

	return nil
}

package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ProducerProposer interface {
	NewProducerProposer(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	AddProposerItem(producerList *quorumpb.BFTProducerBundleItem, originalTrx *quorumpb.Trx, agrmTickCount, agrmTickLength, fromNewEpoch uint64) error
	HandleHBPP(msg *quorumpb.HBMsgv1)
	HandlePPREQ(req *quorumpb.ProducerProposalReq)
}

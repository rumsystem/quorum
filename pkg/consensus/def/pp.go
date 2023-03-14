package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ProducerProposer interface {
	NewProducerProposer(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	AddProposerItem(producerList *quorumpb.BFTProducerBundleItem, originalTrx *quorumpb.Trx)
	HandleChannelMsg(msg *quorumpb.ChannelMsg)
	StartPropose()
	StopPropse()
}

package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type Producer interface {
	NewProducer(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	RecreateBft()
	AddBlock(block *quorumpb.Block) error
	AddTrx(trx *quorumpb.Trx)
	HandleHBMsg(hb *quorumpb.HBMsgv1) error
	StartPropose()
	StopPropose()
}

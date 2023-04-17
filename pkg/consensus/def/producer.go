package def

import (
	"context"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type Producer interface {
	NewProducer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	StartPropose()
	StopPropose()
	AddBlock(block *quorumpb.Block) error
	AddTrx(trx *quorumpb.Trx)
	HandleHBMsg(hb *quorumpb.HBMsgv1) error
}

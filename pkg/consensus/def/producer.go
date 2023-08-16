package def

import (
	"context"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type Producer interface {
	NewProducer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	StartPropose()
	StopPropose()
	AddTrxToTxBuffer(trx *quorumpb.Trx)
	HandleBftMsg(hb *quorumpb.BftMsg) error
}

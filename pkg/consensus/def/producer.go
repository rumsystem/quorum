package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type Producer interface {
	Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	AddTrx(trx *quorumpb.Trx)
	AddProducedBlock(trx *quorumpb.Trx) error
}

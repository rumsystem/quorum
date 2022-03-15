package chain

import quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"

type Snapshot interface {
	Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	SetInterval(sec int)
	Start() error
	Stop() error
	GetSnapshot() ([]*quorumpb.Trx, error)
}

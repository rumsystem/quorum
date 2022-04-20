package chain

import quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

type SnapshotSender interface {
	Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	SetInterval(sec int)
	Start() error
	Stop() error
}

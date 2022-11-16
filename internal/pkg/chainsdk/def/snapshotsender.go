package def

import quorumpb "github.com/rumsystem/quorum/pkg/pb"

type SnapshotSender interface {
	//Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	Init(item *quorumpb.GroupItem, nodename string)
	SetInterval(sec int)
	Start() error
	Stop() error
}

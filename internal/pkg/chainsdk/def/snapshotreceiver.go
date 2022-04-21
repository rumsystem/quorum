package def

import quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

type SnapshotReceiver interface {
	//Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	Init(item *quorumpb.GroupItem, nodename string)
	VerifySignature(s *quorumpb.Snapshot) (bool, error)
	ApplySnapshot(s *quorumpb.Snapshot) error
	GetTag() *quorumpb.SnapShotTag
}

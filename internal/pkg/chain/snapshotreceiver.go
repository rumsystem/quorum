package chain

import quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"

type SnapshotReceiver interface {
	Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	VerifySignature(s *quorumpb.Snapshot) (bool, error)
	ApplySnapshot(s *quorumpb.Snapshot) error
	GetTag() *quorumpb.SnapShotTag
}

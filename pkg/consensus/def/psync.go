package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type PSync interface {
	NewPSyncer(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	RecreateBft()
	TryPropose()
	HandleHBMsg(msg *quorumpb.HBMsgv1) error
}

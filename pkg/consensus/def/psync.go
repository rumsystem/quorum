package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type PSync interface {
	NewPSync(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	RecreateBft()
	AddConsensusReq(req *quorumpb.ConsensusMsg) error
	HandleHBMsg(msg *quorumpb.HBMsgv1) error
}

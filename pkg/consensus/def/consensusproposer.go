package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ConsensusProposer interface {
	NewConsensusProposer(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	HandleHBMsg(msg *quorumpb.HBMsgv1) error
	HandleCCReq(req *quorumpb.ChangeConsensusReq) error
	StartChangeConsensus(producers []string, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error
}

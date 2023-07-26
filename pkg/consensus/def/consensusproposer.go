package def

import (
	"context"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ConsensusProposer interface {
	NewConsensusProposer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	HandleCCMsg(req *quorumpb.CCMsg) error
	ReqChangeConsensus(producers []string, agrmTickLen, agrmTickCnt, fromBlock, fromEpoch, epoch uint64) (string, uint64, error)
	StopAllTasks()
}

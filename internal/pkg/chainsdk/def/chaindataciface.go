package def

import (
	"github.com/libp2p/go-libp2p/core/network"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ChainDataSyncIface interface {
	HandlePsConnMessage(pkg *quorumpb.Package) error
	HandleTrxPsConn(trx *quorumpb.Trx) error
	HandleBlockPsConn(block *quorumpb.Block) error
	HandleTrxRex(trx *quorumpb.Trx, fromstream network.Stream) error
	HandleBlockRex(block *quorumpb.Block, fromstream network.Stream) error
	HandleHBPsConn(hb *quorumpb.HBMsgv1) error
	HandleHBRex(hb *quorumpb.HBMsgv1) error
	HandlePSyncConsesusPsConn(c *quorumpb.ConsensusMsg) error
	HandleConsesusRex(c *quorumpb.ConsensusMsg) error
	StartSync() error
	StopSync()
	IsSyncerIdle() bool
	GetCurrEpoch() int64
}

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
	HandlePSyncPsConn(c *quorumpb.PSyncMsg) error
	HandlePSyncRex(c *quorumpb.PSyncMsg) error
	StartSync() error
	StopSync()
	GetCurrBlock() uint64
}

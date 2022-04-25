package def

import (
	"github.com/libp2p/go-libp2p-core/network"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type ChainDataHandlerIface interface {
	HandleTrxPsConn(trx *quorumpb.Trx) error
	HandleBlockPsConn(block *quorumpb.Block) error
	HandleTrxRex(trx *quorumpb.Trx, fromstream network.Stream) error
	HandleBlockRex(block *quorumpb.Block, fromstream network.Stream) error
	HandleSnapshotPsConn(snapshot *quorumpb.Snapshot) error
}

package iface

import (
	"github.com/libp2p/go-libp2p-core/peer"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type ChainDataHandlerIface interface {
	HandleTrxPsConn(trx *quorumpb.Trx) error
	HandleBlockPsConn(block *quorumpb.Block) error
	HandleSnapshotPsConn(snapshot *quorumpb.Snapshot) error
	HandleTrxRex(trx *quorumpb.Trx, from peer.ID) error
	HandleBlockRex(block *quorumpb.Block, from peer.ID) error
}

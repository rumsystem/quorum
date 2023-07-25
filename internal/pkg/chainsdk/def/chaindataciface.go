package def

import (
	"github.com/libp2p/go-libp2p/core/network"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ChainDataSyncIface interface {
	HandlePsConnMessage(pkg *quorumpb.Package) error
	HandleTrxPsConn(trx *quorumpb.Trx) error
	HandleBlockPsConn(block *quorumpb.Block) error
	HandleBftMsgPsConn(hb *quorumpb.BftMsg) error
	HandleCCMsgPsConn(req *quorumpb.CCMsg) error
	HandleBroadcastMsgPsConn(c *quorumpb.BroadcastMsg) error
	HandleSyncMsgRex(syncMsg *quorumpb.SyncMsg, fromstream network.Stream) error
	StartSync() error
	StopSync()
	GetCurrBlockId() uint64
}

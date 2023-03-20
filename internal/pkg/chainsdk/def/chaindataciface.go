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
	HandleHBPTPsConn(hb *quorumpb.HBMsgv1) error
	HandleHBPCPsConn(hb *quorumpb.HBMsgv1) error
	HandleHBRex(hb *quorumpb.HBMsgv1) error
	HandleChangeConsensusReqPsConn(req *quorumpb.ChangeConsensusReq) error
	HandleGroupBroadcastPsConn(c *quorumpb.GroupBroadcast) error
	StartSync() error
	StopSync()
	GetCurrBlockId() uint64
}

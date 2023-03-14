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
	HandleHBPPPsConn(hb *quorumpb.HBMsgv1) error
	HandleHBPTPsConn(hb *quorumpb.HBMsgv1) error
	HandleHBRex(hb *quorumpb.HBMsgv1) error
	HandlePPReqPsConn(req *quorumpb.ProducerProposalReq) error
	HandleGroupBroadcastPsConn(c *quorumpb.GroupBroadcast) error
	StartSync() error
	StopSync()
	GetCurrBlockId() uint64
}

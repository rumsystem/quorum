package chain

import (
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type Producer interface {
	Init(grp *Group, trxMgr map[string]*TrxMgr, nodeName string)
	AddTrx(trx *quorumpb.Trx)
	AddBlockToPool(block *quorumpb.Block)
	GetBlockForward(trx *quorumpb.Trx) error
	GetBlockBackward(trx *quorumpb.Trx) error
	AddProducedBlock(trx *quorumpb.Trx) error
}

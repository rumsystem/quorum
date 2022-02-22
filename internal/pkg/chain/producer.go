package chain

import (
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type Producer interface {
	Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	AddTrx(trx *quorumpb.Trx)
	AddBlockToPool(block *quorumpb.Block)
	GetBlockForward(trx *quorumpb.Trx) (string, []*quorumpb.Block, bool, error)
	GetBlockBackward(trx *quorumpb.Trx) (string, *quorumpb.Block, bool, error)
	AddProducedBlock(trx *quorumpb.Trx) error
	AddBlock(block *quorumpb.Block) error
}

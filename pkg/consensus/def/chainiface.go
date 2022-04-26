package def

import (
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type ChainMolassesIface interface {
	GetChainSyncIface() chaindef.ChainSyncIface
	GetTrxFactory() chaindef.TrxFactoryIface
	UpdChainInfo(height int64, blockId string) error
	TrxEnqueue(groupId string, trx *quorumpb.Trx) error
	CreateConsensus() error
	RecalChainHeight(blocks []*quorumpb.Block, currentHeight int64, currentHighestBlock *quorumpb.Block, nodename string) (int64, string, error)
	GetTrimedBlocks(blocks []*quorumpb.Block, nodename string) ([]string, error)
	GetMyTrxs(blockIds []string, nodename string, userSignPubkey string) ([]*quorumpb.Trx, error)
	ApplyUserTrxs(trxs []*quorumpb.Trx, nodename string) error
	ApplyProducerTrxs(trxs []*quorumpb.Trx, nodename string) error
}

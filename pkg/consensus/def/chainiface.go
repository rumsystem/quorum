package def

import (
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type ChainMolassesIface interface {
	GetChainSyncIface() chaindef.ChainDataSyncIface
	GetPubqueueIface() chaindef.PublishQueueIface
	GetTrxFactory() chaindef.TrxFactoryIface
	UpdChainInfo(epoch int64) error
	ApplyUserTrxs(trxs []*quorumpb.Trx, nodename string) error
	AddBlock(block *quorumpb.Block) error
	//GetMyTrxs(blockIds []string, nodename string, userSignPubkey string) ([]*quorumpb.Trx, error)
	//RecalChainHeight(blocks []*quorumpb.Block, currentHeight int64, currentHighestBlock *quorumpb.Block, nodename string) (int64, string, error)
	//GetTrimedBlocks(blocks []*quorumpb.Block, nodename string) ([]string, error)
}

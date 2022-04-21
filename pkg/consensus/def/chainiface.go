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
	UpdProducerList()
	UpdUserList()
	CreateConsensus() error
}

package def

import (
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ChainMolassesIface interface {
	GetPubqueueIface() chaindef.PublishQueueIface
	GetTrxFactory() chaindef.TrxFactoryIface
	SaveChainInfoToDb() error
	ApplyTrxsFullNode(trxs []*quorumpb.Trx, nodename string) error
	ApplyTrxsProducerNode(trxs []*quorumpb.Trx, nodename string) error
	SetCurrEpoch(currEpoch uint64)
	IncCurrEpoch()
	GetCurrEpoch() uint64
	SetCurrBlockId(currBlock uint64)
	IncCurrBlockId()
	GetCurrBlockId() uint64
	SetLastUpdate(lastUpdate int64)
	GetLastUpdate() int64
	IsProducer() bool
}

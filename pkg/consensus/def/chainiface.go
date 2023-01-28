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
	SetCurrEpoch(currEpoch int64)
	IncCurrEpoch()
	DecrCurrEpoch()
	GetCurrEpoch() int64
	SetLastUpdate(lastUpdate int64)
	GetLastUpdate() int64
}

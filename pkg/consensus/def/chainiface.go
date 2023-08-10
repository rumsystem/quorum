package def

import (
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ChainMolassesIfaceRumLite interface {
	GetTrxFactory() chaindef.TrxFactoryIface
	SaveChainInfoToDb() error
	ApplyTrxs(trxs []*quorumpb.Trx, nodename string) error
	Sign(hash []byte, pubkey string) ([]byte, error)
	VerifySign(hash, signature []byte, pubkey string) (bool, error)
	HasKey(pubkey string) bool

	//ApplyTrxsProducerNode(trxs []*quorumpb.Trx, nodename string) error
	//SetCurrEpoch(currEpoch uint64)
	//IncCurrEpoch()
	//GetCurrEpoch() uint64
	//SetCurrBlockId(currBlock uint64)
	//IncCurrBlockId()
	//GetCurrBlockId() uint64
	//SetLastUpdate(lastUpdate int64)
	//GetLastUpdate() int64
	//IsProducer() bool
	//IsOwner() bool
	//ReqConsensusChangeDone(bundle *quorumpb.ChangeConsensusResultBundle)
}

type ChainMolassesIface interface {
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
	IsOwner() bool
	VerifySign(hash, signature []byte, pubkey string) (bool, error)
	ReqConsensusChangeDone(bundle *quorumpb.ChangeConsensusResultBundle)
}

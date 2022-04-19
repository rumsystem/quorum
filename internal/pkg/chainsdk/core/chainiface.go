package chain

type ChainMolassesIface interface {
	GetChainCtx() *Chain
	GetTrxFactory() *TrxFactory
	UpdChainInfo(height int64, blockId string) error
	UpdProducerList()
	UpdUserList()
	CreateConsensus() error
}

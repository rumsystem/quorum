package chain

type ChainMolassesIface interface {
	GetUserTrxMgr() *TrxMgr
	GetProducerTrxMgr() *TrxMgr
	UpdChainInfo(height int64, blockId []string) error
	UpdProducerList()
	UpdProducer()
}

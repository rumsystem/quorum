package chain

type ChainMolassesIface interface {
	GetChainCtx() *Chain
	GetTrxFactory() *TrxFactory
	UpdChainInfo(height int64, blockId string) error
	UpdProducerList()
	UpdUserList()
	CreateConsensus() error
	//AskPeerId() error
	//IsSyncerReady() bool
	//SyncBackward(block *quorumpb.Block) error
	//InitSession(channelId string) error
	//GetUserTrxMgr() *TrxMgr
	//GetProducerTrxMgr() *TrxMgr
}

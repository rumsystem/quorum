package def

type ChainSyncIface interface {
	SyncBackward(blockId string, nodename string) error
	SyncForward(blockId string, nodename string) error
	StopSync() error
	IsSyncerIdle() bool
}

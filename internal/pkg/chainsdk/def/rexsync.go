package def

type RexSyncResult struct {
	Provider              string
	FromBlock             uint64
	BlockProvided         int32
	SyncResult            string
	LastSyncTaskTimestamp int64
}

package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GroupIface interface {
	SendRawTrx(trx *quorumpb.Trx) (string, error)
	GetTrx(trxId string) (*quorumpb.Trx, error)
	GetTrxFromCache(trxId string) (*quorumpb.Trx, error)
	GetRexSyncerStatus() string
	StartSync(restart bool) error
	StopSync() error
}

type RexSyncResult struct {
	Provider              string
	FromBlock             uint64
	BlockProvided         int32
	SyncResult            string
	LastSyncTaskTimestamp int64
	NextSyncTaskTimeStamp int64
}

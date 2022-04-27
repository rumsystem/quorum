package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type GroupIface interface {
	GetTrx(trxId string) (*quorumpb.Trx, []int64, error)
	GetTrxFromCache(trxId string) (*quorumpb.Trx, []int64, error)
	GetSyncerStatus() int8
}

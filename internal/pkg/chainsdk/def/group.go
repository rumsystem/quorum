package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type GroupIface interface {
	SendRawTrx(trx *quorumpb.Trx) (string, error)
	GetTrx(trxId string) (*quorumpb.Trx, []int64, error)
	GetTrxFromCache(trxId string) (*quorumpb.Trx, []int64, error)
	GetSyncerStatus() int8
}

package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type TrxStorageType uint

const (
	Chain TrxStorageType = iota
	Cache
)

type ChainStorageIface interface {
	DeleteRelay(relayid string) (bool, *quorumpb.GroupRelayItem, error)
	AddRelayActivity(groupRelayItem *quorumpb.GroupRelayItem) (string, error)
	AddRelayReq(groupRelayItem *quorumpb.GroupRelayItem) (string, error)
}

type TrxStorageIface interface {
	GetTrx(trxId string, storagetype TrxStorageType, prefix ...string) (t *quorumpb.Trx, n []int64, err error)
}

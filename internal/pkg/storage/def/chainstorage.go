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

type APIHandlerIface interface {
	IsUserAnnounced(groupId, userSignPubkey string, prefix ...string) (bool, error)
	GetSendTrxAuthListByGroupId(groupId string, listType quorumpb.AuthListType, prefix ...string) ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error)
}

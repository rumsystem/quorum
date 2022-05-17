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
	IsProducerAnnounced(groupId, producerSignPubkey string, prefix ...string) (bool, error)
	GetSendTrxAuthListByGroupId(groupId string, listType quorumpb.AuthListType, prefix ...string) ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error)
	GetTrxAuthModeByGroupId(groupId string, trxType quorumpb.TrxType, prefix ...string) (quorumpb.TrxAuthMode, error)
	GetAnnounceProducersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error)
	GetAnnounceUsersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error)
	GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error)
}

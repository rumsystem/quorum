package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type TrxStorageType uint

const (
	Chain TrxStorageType = iota
	Cache
)

type TrxStorageIface interface {
	GetTrx(groupId string, trxId string, storagetype TrxStorageType, prefix ...string) (t *quorumpb.Trx, err error)
}

type APIHandlerIface interface {
	IsUserAnnounced(groupId, userSignPubkey string, prefix ...string) (bool, error)
	IsProducerAnnounced(groupId, producerSignPubkey string, prefix ...string) (bool, error)
	IsUser(groupId, userSignPubkey string, prefix ...string) (bool, error)
	IsProducer(groupId, producerSignPubkey string, prefix ...string) (bool, error)
	GetSendTrxAuthListByGroupId(groupId string, listType quorumpb.AuthListType, prefix ...string) ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error)
	GetTrxAuthModeByGroupId(groupId string, trxType quorumpb.TrxType, prefix ...string) (quorumpb.TrxAuthMode, error)
	GetAnnouncedProducers(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error)
	GetAnnouncedUsers(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error)
	GetAnnouncedProducer(groupId, producerSignPubkey string, prefix ...string) (*quorumpb.AnnounceItem, error)
	GetAnnouncedUser(groupId, userSignPubkey string, prefix ...string) (*quorumpb.AnnounceItem, error)
	GetUsers(groupId string, prefix ...string) ([]*quorumpb.UserItem, error)
	GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error)
	GetUser(groupId, userSignPubkey string, prefix ...string) (*quorumpb.UserItem, error)
	GetProducer(groupId, producerSignPubkey string, prefix ...string) (*quorumpb.ProducerItem, error)
	GetAllChangeConsensusResult(groupId string, prefix ...string) ([]*quorumpb.ChangeConsensusResultBundle, error)
	GetChangeConsensusResultByReqId(groupId, reqId string, prefix ...string) (*quorumpb.ChangeConsensusResultBundle, error)
}

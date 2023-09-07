package storage

import (
	"fmt"
	"strconv"

	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

const (
	TRX_PREFIX             = "trx"             //trx
	BLK_PREFIX             = "blk"             //block
	GRP_PREFIX             = "grp"             //group
	CHNINFO_PREFIX         = "chain"           //chaininfo
	CNT_PREFIX             = "cnt"             //content
	PRD_PREFIX             = "prd"             //producer
	SYNCER_PREFIX          = "syncer"          //syncer
	CHD_PREFIX             = "chd"             //cached
	APP_CONFIG_PREFIX      = "app_conf"        //group configuration
	CHAIN_CONFIG_PREFIX    = "chn_conf"        //chain configuration
	TRX_AUTH_TYPE_PREFIX   = "trx_auth"        //trx auth type
	ALLW_LIST_PREFIX       = "alw_list"        //allow list
	DENY_LIST_PREFIX       = "dny_list"        //deny list
	PRD_TRX_ID_PREFIX      = "prd_trxid"       //trxid of latest trx which update group producer list
	DSCHD_PREFIX           = "dschd"           //cache datasource
	CONSENSUS_NONCE_PREFIX = "consensus_nonce" //group consensus nonce

	// groupinfo db
	GROUPITEM_PREFIX = "grpitem"
	GROUPSEED_PREFIX = "grpseed"
	RELAY_PREFIX     = "rly" //relay

	// consensus db
	BUFD_TRX        = "bfd_trx" //buffered trx (used by acs)
	CNS_INFO_PREFIX = "cns"     //consensus info

	// cellar db
	CELLAR_PREFIX     = "cellar"     //cellar
	CELLAR_GRP_PREFIX = "cellar_grp" //cellar group
)

func _getEthPubkey(libp2pPubkey string) string {
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(libp2pPubkey)
	if pk == "" {
		pk = libp2pPubkey
	}

	return pk
}

func GetBlockPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + BLK_PREFIX + "_"
	if groupId != "" {
		key = key + groupId + "_"
	}
	return key
}

func GetBlockKey(groupId string, blockID uint64, prefix ...string) string {
	epochSD := strconv.FormatUint(blockID, 10)
	_prefix := GetBlockPrefix(groupId, prefix...)
	return _prefix + epochSD
}

func GetCachedBlockPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_"
	if groupId != "" {
		return key + groupId + "_"
	}
	return key
}

func GetDSCachedBlockPrefix(groupId string, blockId uint64, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	blockIdstr := strconv.FormatUint(blockId, 10)
	key := nodeprefix + DSCHD_PREFIX + "_" + BLK_PREFIX + "_"
	if groupId != "" {
		return key + groupId + "_" + blockIdstr
	}
	return key
}

func GetCachedBlockKey(groupId string, blockId uint64, prefix ...string) string {
	epochSD := strconv.FormatUint(blockId, 10)
	_prefix := GetCachedBlockPrefix(groupId, prefix...)
	return _prefix + epochSD
}

func GetGroupItemPrefix() string {
	return GROUPITEM_PREFIX + "_"
}

func GetGroupItemKey(groupId string) string {
	return GetGroupItemPrefix() + groupId
}

func GetChainInfoEpoch(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + CHNINFO_PREFIX + "_" + groupId + "_" + "currepoch"
}

func GetChainInfoLastUpdate(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + CHNINFO_PREFIX + "_" + groupId + "_" + "lastupdate"
}

func GetChainInfoBlock(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + CHNINFO_PREFIX + "_" + groupId + "_" + "currblock"
}

func GetPostPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + groupId
}

func GetPostKey(groupId string, timestamp string, trxid string, prefix ...string) string {
	_prefix := GetPostPrefix(groupId, prefix...)
	return _prefix + "_" + timestamp + "_" + trxid
}

func GetProducerPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + PRD_PREFIX + "_" + groupId + "_"
}

func GetProducerKey(groupId string, pk string, prefix ...string) string {
	_prefix := GetProducerPrefix(groupId, prefix...)
	return _prefix + pk
}

func GetSyncerPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + SYNCER_PREFIX + "_" + groupId + "_"
}

func GetSyncerKey(groupId string, pubkey string, prefix ...string) string {
	_prefix := GetSyncerPrefix(groupId, prefix...)
	pk := _getEthPubkey(pubkey)
	return _prefix + pk
}

func GetChainConfigPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId
}

func _getChainConfigKey(groupId string, _type string, item string, prefix ...string) string {
	_prefix := GetChainConfigPrefix(groupId, prefix...)
	return _prefix + "_" + _type + "_" + item
}

func GetChainConfigAuthKey(groupId string, _type string, prefix ...string) string {
	return _getChainConfigKey(groupId, TRX_AUTH_TYPE_PREFIX, _type, prefix...)
}

func GetChainConfigAllowPrefix(groupId string, prefix ...string) string {
	_prefix := GetChainConfigPrefix(groupId, prefix...)
	return _prefix + "_" + ALLW_LIST_PREFIX
}

func GetChainConfigAllowKey(groupId string, pk string, prefix ...string) string {
	return _getChainConfigKey(groupId, ALLW_LIST_PREFIX, pk, prefix...)
}

func GetChainConfigDenyKey(groupId string, pk string, prefix ...string) string {
	return _getChainConfigKey(groupId, DENY_LIST_PREFIX, pk, prefix...)
}

func GetChainConfigDenyPrefix(groupId string, prefix ...string) string {
	_prefix := GetChainConfigPrefix(groupId, prefix...)
	return _prefix + "_" + DENY_LIST_PREFIX
}

func GetAppConfigPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + APP_CONFIG_PREFIX + "_" + groupId
}

func GetAppConfigKey(groupId string, name string, prefix ...string) string {
	_prefix := GetAppConfigPrefix(groupId, prefix...)
	return _prefix + "_" + name
}

func GetConsensusNonceKey(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + CONSENSUS_NONCE_PREFIX + "_" + groupId
}

func GetProducerTrxIDKey(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + PRD_TRX_ID_PREFIX + "_" + groupId
}
func GetTrxPrefix(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + groupId + "_"
	return key
}

func GetTrxKey(groupId, trxId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + groupId + "_"
	if trxId != "" {
		key = key + trxId
	}
	return key
}

func GetSeedKey(groupID string) []byte {
	return []byte(fmt.Sprintf("%s_%s", GROUPSEED_PREFIX, groupID))
}

func GetTrxHBBPrefix(queueId string) string {
	return BUFD_TRX + "_" + queueId + "_"
}

func GetTrxHBBKey(queueId string, trxId string) string {
	prefix := GetTrxHBBPrefix(queueId)
	return prefix + trxId
}

// Relay
func GetRelayPrefix() string {
	return RELAY_PREFIX
}

func GetRelayReqPrefix() string {
	return GetRelayPrefix() + "_req_"
}

func GetRelayReqKey(groupId string, _type string) string {
	return GetRelayReqPrefix() + groupId + "_" + _type
}

func GetRelayReqUserKey(groupId string, _type string, pubkey string) string {
	return GetRelayReqKey(groupId, _type) + "_" + pubkey
}

func GetRelayActivityKey(groupId, _type string) string {
	return GetRelayPrefix() + "_activity_" + groupId + "_" + _type
}

func GetRelayApprovedKey(groupId, _type string) string {
	return GetRelayPrefix() + "_approved_" + groupId + "_" + _type
}

func GetGroupConsensusKey(groupId string, prefix ...string) string {
	nodeprefix := utils.GetPrefix(prefix...)
	return nodeprefix + CNS_INFO_PREFIX + "_" + groupId + "_"
}

func GetCellarPrefix() string {
	return CELLAR_PREFIX + "_"
}

func GetCellarGroupPrefix() string {
	return CELLAR_GRP_PREFIX + "_"
}

func GetCellarSeedKey(cellarId string) string {
	return GetCellarPrefix() + "seed_" + cellarId
}

func GetCellarKey(cellarId string) string {
	result := GetCellarPrefix() + cellarId
	return result
}

func GetCellarGroupKey(cellarId, groupId string) string {
	return GetCellarGroupPrefix() + cellarId + "_" + groupId
}

func GetCellarGroupPrefixByCellarId(cellarId string) string {
	return GetCellarGroupPrefix() + cellarId + "_"
}

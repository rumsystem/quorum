package chainstorage

import (
	"errors"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateChainConfigTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return cs.UpdateChainConfig(trx.Data, prefix...)
}

func (cs *Storage) UpdateChainConfig(data []byte, prefix ...string) (err error) {
	chaindb_log.Infof("UpdateChainConfig called")
	nodeprefix := utils.GetPrefix(prefix...)
	item := &quorumpb.ChainConfigItem{}

	if err := proto.Unmarshal(data, item); err != nil {
		chaindb_log.Infof(err.Error())
		return err
	}

	if item.Type == quorumpb.ChainConfigType_SET_TRX_AUTH_MODE {
		authModeItem := &quorumpb.SetTrxAuthModeItem{}
		if err := proto.Unmarshal(item.Data, authModeItem); err != nil {
			chaindb_log.Infof(err.Error())
			return err
		}

		key := nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + s.TRX_AUTH_TYPE_PREFIX + "_" + authModeItem.Type.String()
		return cs.dbmgr.Db.Set([]byte(key), data)
	} else if item.Type == quorumpb.ChainConfigType_UPD_ALW_LIST ||
		item.Type == quorumpb.ChainConfigType_UPD_DNY_LIST {
		ruleListItem := &quorumpb.ChainSendTrxRuleListItem{}
		if err := proto.Unmarshal(item.Data, ruleListItem); err != nil {
			return err
		}
		pk, _ := localcrypto.Libp2pPubkeyToEthBase64(ruleListItem.Pubkey)
		if pk == "" {
			pk = ruleListItem.Pubkey
		}

		var key string
		if item.Type == quorumpb.ChainConfigType_UPD_ALW_LIST {
			key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + s.ALLW_LIST_PREFIX + "_" + pk
		} else {
			key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + s.DENY_LIST_PREFIX + "_" + pk
		}

		chaindb_log.Infof("key %s", key)

		if ruleListItem.Action == quorumpb.ActionType_ADD {
			return cs.dbmgr.Db.Set([]byte(key), data)
		} else {
			exist, err := cs.dbmgr.Db.IsExist([]byte(key))
			if !exist {
				if err != nil {
					return err
				}
				return errors.New("key Not Found")
			}
		}

		return cs.dbmgr.Db.Delete([]byte(key))
	} else {
		return errors.New("Unsupported ChainConfig type")
	}
}

func (cs *Storage) GetTrxAuthModeByGroupId(groupId string, trxType quorumpb.TrxType, prefix ...string) (quorumpb.TrxAuthMode, error) {
	nodoeprefix := utils.GetPrefix(prefix...)
	key := nodoeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.TRX_AUTH_TYPE_PREFIX + "_" + trxType.String()

	//if not specified by group owner
	//follow deny list by default
	//if in deny list, access prohibit
	//if not in deny list, access granted
	isExist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if !isExist {
		return quorumpb.TrxAuthMode_FOLLOW_DNY_LIST, nil
	}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return -1, err
	}

	chainConfigItem := &quorumpb.ChainConfigItem{}
	if err := proto.Unmarshal(value, chainConfigItem); err != nil {
		return -1, err
	}

	trxAuthitem := quorumpb.SetTrxAuthModeItem{}
	perr := proto.Unmarshal(chainConfigItem.Data, &trxAuthitem)
	if perr != nil {
		return -1, perr
	}

	return trxAuthitem.Mode, nil
}

func (cs *Storage) GetSendTrxAuthListByGroupId(groupId string, listType quorumpb.AuthListType, prefix ...string) ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error) {
	var chainConfigList []*quorumpb.ChainConfigItem
	var sendTrxRuleList []*quorumpb.ChainSendTrxRuleListItem

	nodeprefix := utils.GetPrefix(prefix...)
	var key string
	if listType == quorumpb.AuthListType_ALLOW_LIST {
		key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.ALLW_LIST_PREFIX
	} else {
		key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.DENY_LIST_PREFIX
	}
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		chainConfigItem := quorumpb.ChainConfigItem{}
		err = proto.Unmarshal(v, &chainConfigItem)
		if err != nil {
			return err
		}
		chainConfigList = append(chainConfigList, &chainConfigItem)
		sendTrxRuleListItem := quorumpb.ChainSendTrxRuleListItem{}
		err = proto.Unmarshal(chainConfigItem.Data, &sendTrxRuleListItem)
		if err != nil {
			return err
		}
		sendTrxRuleList = append(sendTrxRuleList, &sendTrxRuleListItem)

		pk, _ := localcrypto.Libp2pPubkeyToEthBase64(sendTrxRuleListItem.Pubkey)
		if pk == "" {
			pk = sendTrxRuleListItem.Pubkey
		}
		chaindb_log.Infof("sendTrx %s", pk)

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return chainConfigList, sendTrxRuleList, nil
}

func (cs *Storage) CheckTrxTypeAuth(groupId, pubkey string, trxType quorumpb.TrxType, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(pubkey)
	if pk == "" {
		pk = pubkey
	}

	keyAllow := nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.ALLW_LIST_PREFIX + "_" + pk
	keyDeny := nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.DENY_LIST_PREFIX + "_" + pk

	isInAllowList, err := cs.dbmgr.Db.IsExist([]byte(keyAllow))
	if err != nil {
		return false, err
	}

	if isInAllowList {
		v, err := cs.dbmgr.Db.Get([]byte(keyAllow))
		chainConfigItem := quorumpb.ChainConfigItem{}
		err = proto.Unmarshal(v, &chainConfigItem)
		if err != nil {
			return false, err
		}

		allowItem := quorumpb.ChainSendTrxRuleListItem{}
		err = proto.Unmarshal(chainConfigItem.Data, &allowItem)
		if err != nil {
			return false, err
		}

		//check if trxType allowed
		for _, allowTrxType := range allowItem.Type {
			if trxType == allowTrxType {
				return true, nil
			}
		}
	}

	isInDenyList, err := cs.dbmgr.Db.IsExist([]byte(keyDeny))
	if err != nil {
		return false, err
	}

	if isInDenyList {
		v, err := cs.dbmgr.Db.Get([]byte(keyDeny))
		chainConfigItem := quorumpb.ChainConfigItem{}
		err = proto.Unmarshal(v, &chainConfigItem)
		if err != nil {
			return false, err
		}

		denyItem := quorumpb.ChainSendTrxRuleListItem{}
		err = proto.Unmarshal(chainConfigItem.Data, &denyItem)
		if err != nil {
			return false, err
		}
		//check if trxType allowed
		for _, denyTrxType := range denyItem.Type {
			if trxType == denyTrxType {
				return false, nil
			}
		}
	}
	trxAuthMode, err := cs.GetTrxAuthModeByGroupId(groupId, trxType, prefix...)
	if err != nil {
		return false, err
	}

	if trxAuthMode == quorumpb.TrxAuthMode_FOLLOW_ALW_LIST {
		//not in allow list, so return false, access denied
		return false, nil
	} else {
		//not in deny list, so return true, access granted
		return true, nil
	}
}

func (cs *Storage) GetAllChainConfigInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_"
	var chainConfigByteList [][]byte

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		chainConfigByteList = append(chainConfigByteList, v)
		return nil
	})

	return chainConfigByteList, err
}

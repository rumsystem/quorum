package chainstorage

/*
import (
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateChainConfig(data []byte, prefix ...string) (err error) {
	chaindb_log.Infof("UpdateChainConfig called")
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

		key := s.GetChainConfigAuthKey(item.GroupId, authModeItem.Type.String(), prefix...)
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
			key = s.GetChainConfigAllowKey(item.GroupId, pk, prefix...)
		} else {
			key = s.GetChainConfigDenyKey(item.GroupId, pk, prefix...)
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
		return errors.New("unsupported ChainConfig type")
	}
}

func (cs *Storage) GetTrxAuthModeByGroupId(groupId string, trxType quorumpb.TrxType, prefix ...string) (quorumpb.TrxAuthMode, error) {
	key := s.GetChainConfigAuthKey(groupId, trxType.String(), prefix...)

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

	var key string
	if listType == quorumpb.AuthListType_ALLOW_LIST {
		key = s.GetChainConfigAllowPrefix(groupId, prefix...)
	} else {
		key = s.GetChainConfigDenyPrefix(groupId, prefix...)
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

func (cs *Storage) CheckPackageTypeAuth(groupId, pubkey string, packageType quorumpb.PackageType, prefix ...string) (bool, error) {
	//tbd implement package type auth

	//current just return true
	return true, nil
}

func (cs *Storage) CheckTrxTypeAuth(groupId, pubkey string, trxType quorumpb.TrxType, prefix ...string) (bool, error) {
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(pubkey)
	if pk == "" {
		pk = pubkey
	}

	keyAllow := s.GetChainConfigAllowKey(groupId, pk, prefix...)
	keyDeny := s.GetChainConfigDenyKey(groupId, pk, prefix...)

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
	key := s.GetChainConfigPrefix(groupId, Prefix...)
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

*/

package chainstorage

import (
	"errors"
	"fmt"
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

type Storage struct {
	dbmgr *s.DbMgr
}

var storage *Storage
var chaindb_log = logging.Logger("chaindb")

func NewChainStorage(dbmgr *s.DbMgr) (storage *Storage) {
	if storage == nil {
		storage = &Storage{dbmgr}
	}
	return storage
}

func (cs *Storage) UpdateAnnounceResult(announcetype quorumpb.AnnounceType, groupId, signPubkey string, result bool, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + announcetype.String() + "_" + signPubkey

	var pAnnounced *quorumpb.AnnounceItem
	pAnnounced = &quorumpb.AnnounceItem{}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(value, pAnnounced)
	if err != nil {
		return err
	}

	if result {
		pAnnounced.Result = quorumpb.ApproveType_APPROVED
	} else {
		pAnnounced.Result = quorumpb.ApproveType_ANNOUNCED
	}

	value, err = proto.Marshal(pAnnounced)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) UpdateAnnounce(data []byte, prefix ...string) (err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + item.GroupId + "_" + item.Type.Enum().String() + "_" + item.SignPubkey
	return cs.dbmgr.Db.Set([]byte(key), data)
}

//save trx
func (cs *Storage) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.TRX_PREFIX + "_" + trx.TrxId + "_" + fmt.Sprint(trx.Nonce)
	value, err := proto.Marshal(trx)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}

//UNUSED
//rm Trx
func (cs *Storage) RmTrx(trxId string, nonce int64, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.TRX_PREFIX + "_" + trxId + "_" + fmt.Sprint(nonce)
	return cs.dbmgr.Db.Delete([]byte(key))
}

func (cs *Storage) UpdTrx(trx *quorumpb.Trx, prefix ...string) error {
	return cs.AddTrx(trx, prefix...)
}

//update group snapshot
func (cs *Storage) UpdateSnapshotTag(groupId string, snapshotTag *quorumpb.SnapShotTag, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SNAPSHOT_PREFIX + "_" + groupId
	value, err := proto.Marshal(snapshotTag)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) GetSnapshotTag(groupId string, prefix ...string) (*quorumpb.SnapShotTag, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SNAPSHOT_PREFIX + "_" + groupId

	//check if item exist
	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("SnapshotTag Not Found")
	}

	snapshotTag := quorumpb.SnapShotTag{}
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &snapshotTag)
	return &snapshotTag, err
}

func (cs *Storage) UpdateSchema(trx *quorumpb.Trx, prefix ...string) (err error) {
	item := &quorumpb.SchemaItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SMA_PREFIX + "_" + item.GroupId + "_" + item.Type

	if item.Action == quorumpb.ActionType_ADD {
		return cs.dbmgr.Db.Set([]byte(key), trx.Data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if item exist
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Announce Not Found")
		}

		return cs.dbmgr.Db.Delete([]byte(key))
	} else {
		err := errors.New("unknow msgType")
		return err
	}
}

//relaystatus: req, approved and activity
func (cs *Storage) AddRelayReq(groupRelayItem *quorumpb.GroupRelayItem) (string, error) {
	groupRelayItem.RelayId = guuid.New().String()
	key := s.RELAY_PREFIX + "_req_" + groupRelayItem.GroupId + "_" + groupRelayItem.Type

	//dbMgr.GroupInfoDb.PrefixDelete([]byte(RELAY_PREFIX))

	if groupRelayItem.Type == "user" {
		key = s.RELAY_PREFIX + "_req_" + groupRelayItem.GroupId + "_" + groupRelayItem.Type + "_" + groupRelayItem.UserPubkey
	}
	//check if group relay req exist
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist { //check if not expire
		return "", errors.New("the same relay req exist ")
	}

	//add group relay req to db
	value, err := proto.Marshal(groupRelayItem)
	if err != nil {
		return "", err
	}
	return groupRelayItem.RelayId, cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) AddRelayActivity(groupRelayItem *quorumpb.GroupRelayItem) (string, error) {
	key := s.RELAY_PREFIX + "_activity_" + groupRelayItem.GroupId + "_" + groupRelayItem.Type
	//check if group relay req exist
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist { //check if not expire
		return "", errors.New("the same relay exist ")
	}

	//add group relay to db
	value, err := proto.Marshal(groupRelayItem)
	if err != nil {
		return "", err
	}
	return groupRelayItem.RelayId, cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) DeleteRelay(relayid string) (bool, *quorumpb.GroupRelayItem, error) {
	key := s.RELAY_PREFIX
	succ := false
	relayitem := quorumpb.GroupRelayItem{}
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		err = proto.Unmarshal(v, &relayitem)
		if err == nil {
			if relayitem.RelayId == relayid {
				err = cs.dbmgr.GroupInfoDb.Delete(k)
				if err == nil {
					succ = true
				}
			}
		}
		return nil
	})
	return succ, &relayitem, err
}

func (cs *Storage) ApproveRelayReq(reqid string) (bool, *quorumpb.GroupRelayItem, error) {
	key := s.RELAY_PREFIX + "_req_"
	succ := false

	relayreq := quorumpb.GroupRelayItem{}
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		err = proto.Unmarshal(v, &relayreq)
		if relayreq.RelayId == reqid {
			relayreq.ApproveTime = time.Now().UnixNano()
			approvedkey := s.RELAY_PREFIX + "_approved_" + relayreq.GroupId + "_" + relayreq.Type
			approvedvalue, err := proto.Marshal(&relayreq)
			if err != nil {
				return err
			}
			err = cs.dbmgr.GroupInfoDb.Set([]byte(approvedkey), approvedvalue)
			if err != nil {
				return err
			}
			succ = true
			return cs.dbmgr.GroupInfoDb.Delete(k)
		}
		return nil
	})
	return succ, &relayreq, err
}

func (cs *Storage) GetRelay(relaystatus string, groupid string) ([]*quorumpb.GroupRelayItem, error) {
	switch relaystatus {
	case "req", "approved", "activity":
		key := s.RELAY_PREFIX + "_" + relaystatus + "_" + groupid
		groupRelayItemList := []*quorumpb.GroupRelayItem{}
		err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			relayreq := quorumpb.GroupRelayItem{}
			err = proto.Unmarshal(v, &relayreq)
			groupRelayItemList = append(groupRelayItemList, &relayreq)
			return nil
		})
		return groupRelayItemList, err
	}
	return nil, errors.New("unknown relaystatus")
}

func (cs *Storage) GetRelayReq(groupid string) ([]*quorumpb.GroupRelayItem, error) {
	return cs.GetRelay("req", groupid)
}

func (cs *Storage) GetRelayApproved(groupid string) ([]*quorumpb.GroupRelayItem, error) {
	return cs.GetRelay("approved", groupid)
}

func (cs *Storage) GetRelayActivity(groupid string) ([]*quorumpb.GroupRelayItem, error) {
	return cs.GetRelay("activity", groupid)
}

func (cs *Storage) GetBlockHeight(blockId string, prefix ...string) (int64, error) {
	pChunk, err := cs.dbmgr.GetBlockChunk(blockId, false, prefix...)
	if err != nil {
		return -1, err
	}
	return pChunk.Height, nil
}

func (cs *Storage) GetAllSchemasByGroup(groupId string, prefix ...string) ([]*quorumpb.SchemaItem, error) {
	var scmList []*quorumpb.SchemaItem

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SMA_PREFIX + "_" + groupId

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.SchemaItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		scmList = append(scmList, &item)
		return nil
	})

	return scmList, err
}

func (cs *Storage) GetUsers(groupId string, prefix ...string) ([]*quorumpb.UserItem, error) {
	var pList []*quorumpb.UserItem
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.USR_PREFIX + "_" + groupId

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.UserItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		pList = append(pList, &item)
		return nil
	})
	return pList, err
}

func (cs *Storage) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var pList []*quorumpb.ProducerItem
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.PRD_PREFIX + "_" + groupId

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.ProducerItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		pList = append(pList, &item)
		return nil
	})
	return pList, err
}

func (cs *Storage) GetAnnounceProducersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String()
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.AnnounceItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		aList = append(aList, &item)
		return nil
	})

	return aList, err
}

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

		var key string
		if item.Type == quorumpb.ChainConfigType_UPD_ALW_LIST {
			key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + s.ALLW_LIST_PREFIX + "_" + ruleListItem.Pubkey
		} else {
			key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + s.DENY_LIST_PREFIX + "_" + ruleListItem.Pubkey
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

func (cs *Storage) AddPost(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.GRP_PREFIX + "_" + s.CNT_PREFIX + "_" + trx.GroupId + "_" + fmt.Sprint(trx.TimeStamp) + "_" + trx.TrxId
	chaindb_log.Debugf("Add POST with key %s", key)

	var ctnItem *quorumpb.PostItem
	ctnItem = &quorumpb.PostItem{}

	ctnItem.TrxId = trx.TrxId
	ctnItem.PublisherPubkey = trx.SenderPubkey
	ctnItem.Content = trx.Data
	ctnItem.TimeStamp = trx.TimeStamp
	ctnBytes, err := proto.Marshal(ctnItem)
	if err != nil {
		return err
	}

	return cs.dbmgr.Db.Set([]byte(key), ctnBytes)
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

		chaindb_log.Infof("sendTrx %s", sendTrxRuleListItem.Pubkey)

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return chainConfigList, sendTrxRuleList, nil
}

func (cs *Storage) CheckTrxTypeAuth(groupId, pubkey string, trxType quorumpb.TrxType, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)

	keyAllow := nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.ALLW_LIST_PREFIX + "_" + pubkey
	keyDeny := nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + s.DENY_LIST_PREFIX + "_" + pubkey

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

//func (cs *Storage) IsProducer(groupId, producerPubKey string, prefix ...string) (bool, error) {
//	nodeprefix := utils.GetPrefix(prefix...)
//	key := nodeprefix + s.PRD_PREFIX + "_" + groupId + "_" + producerPubKey
//
//	//check if group exist
//	return cs.dbmgr.Db.IsExist([]byte(key))
//}

//func (cs *Storage) GetSchemaByGroup(groupId, schemaType string, prefix ...string) (*quorumpb.SchemaItem, error) {
//	nodeprefix := utils.GetPrefix(prefix...)
//	key := nodeprefix + s.SMA_PREFIX + "_" + groupId + "_" + schemaType
//
//	schema := quorumpb.SchemaItem{}
//	value, err := cs.dbmgr.Db.Get([]byte(key))
//	if err != nil {
//		return nil, err
//	}
//
//	err = proto.Unmarshal(value, &schema)
//	if err != nil {
//		return nil, err
//	}
//
//	return &schema, err
//}

package chainstorage

import (
	"encoding/json"
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) AddGroup(parentGroupId string, groupItem *quorumpb.GroupItem) error {
	//check if group exist
	key := s.GetGroupItemKey(parentGroupId, groupItem.GroupId)
	exist, _ := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist {
		return errors.New("group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) UpdGroup(parentGroupId string, groupItem *quorumpb.GroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	key := s.GetGroupItemKey(parentGroupId, groupItem.GroupId)
	//upd group to db
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) RmGroup(parentGroupId string, groupId string) error {
	//check if group exist
	key := s.GetGroupItemKey(parentGroupId, groupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("group Not found")
	}

	//delete group
	return cs.dbmgr.GroupInfoDb.Delete([]byte(key))
}

func (cs *Storage) GetGroupItem(parentGroupId, groupId string) (*quorumpb.GroupItem, error) {
	//check if group exist
	key := s.GetGroupItemKey(parentGroupId, groupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("group not found")
	}

	bGrpInfo, err := cs.dbmgr.GroupInfoDb.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	grpInfo := &quorumpb.GroupItem{}
	err = proto.Unmarshal(bGrpInfo, grpInfo)
	if err != nil {
		return nil, err
	}

	return grpInfo, nil
}

func (cs *Storage) RemoveGroupData(groupId string, prefix ...string) error {
	return RemoveGroupData(cs.dbmgr.Db, groupId, prefix...)
}

func RemoveGroupData(db s.QuorumStorage, groupId string, prefix ...string) error {
	var keys []string

	//remove all group POST
	key := s.GetPostPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group producers
	key = s.GetProducerPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group syncers
	key = s.GetSyncerPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group chain_config item
	key = s.GetChainConfigPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group app_config item
	key = s.GetAppConfigPrefix(groupId, prefix...)
	keys = append(keys, key)

	//nonce prefix
	key = s.GetConsensusNonceKey(groupId, prefix...)
	keys = append(keys, key)

	//trx_id for producer update trx
	key = s.GetProducerTrxIDKey(groupId, prefix...)
	keys = append(keys, key)

	// cached block
	key = s.GetCachedBlockPrefix(groupId, prefix...)
	keys = append(keys, key)

	// block
	key = s.GetBlockPrefix(groupId, prefix...)
	keys = append(keys, key)

	// trx
	key = s.GetTrxPrefix(groupId, prefix...)
	keys = append(keys, key)

	//remove all
	for _, key_prefix := range keys {
		_, err := db.PrefixDelete([]byte(key_prefix))
		if err != nil {
			return err
		}
	}

	//delete group seed
	seedKey := s.GetSeedKey(groupId)
	db.Delete([]byte(seedKey))

	return nil
}

// NodeSDK
func (cs *Storage) AddGroupV2(parentGroupId string, groupItem *quorumpb.NodeSDKGroupItem) error {
	//check if group exist
	key := s.GetGroupItemKey(parentGroupId, groupItem.Group.GroupId)
	exist, _ := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist {
		return errors.New("group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

// Get Gorup Info
func (cs *Storage) GetGroupInfoV2(parentGroupId string, groupId string) (*quorumpb.NodeSDKGroupItem, error) {
	key := s.GetGroupItemKey(parentGroupId, groupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("group Not Found")
	}

	groupInfoByte, err := cs.dbmgr.GroupInfoDb.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	groupInfo := &quorumpb.NodeSDKGroupItem{}
	err = proto.Unmarshal(groupInfoByte, groupInfo)
	if err != nil {
		return nil, err
	}

	return groupInfo, nil
}

func (cs *Storage) GetAllGroupsV2(parentGroupId string) ([]*quorumpb.NodeSDKGroupItem, error) {
	var result []*quorumpb.NodeSDKGroupItem

	key := s.GetGroupItemKey(parentGroupId, "")
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := &quorumpb.NodeSDKGroupItem{}
		err = proto.Unmarshal(v, item)
		if err != nil {
			return err
		}
		result = append(result, item)
		return nil
	})
	return result, err
}

func (cs *Storage) UpdGroupV2(parentGroupId string, groupItem *quorumpb.NodeSDKGroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	key := s.GetGroupItemKey(parentGroupId, groupItem.Group.GroupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("group is not existed")
	}

	//upd group to db
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

// group seed
func (cs *Storage) SetGroupSeed(seed *quorumpb.GroupSeed) error {
	key := s.GetSeedKey(seed.GenesisBlock.GroupId)
	value, err := proto.Marshal(seed)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) GetGroupSeed(groupId string) (*quorumpb.GroupSeed, error) {
	key := s.GetSeedKey(groupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("group seed not exist")
	}

	value, err := cs.dbmgr.GroupInfoDb.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var result quorumpb.GroupSeed
	if err := proto.Unmarshal(value, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (cs *Storage) GetAllGroupSeeds() (map[string]*quorumpb.GroupSeed, error) {
	var seeds map[string]*quorumpb.GroupSeed = make(map[string]*quorumpb.GroupSeed)

	prefix := s.GetSeedPrefix()
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(prefix), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		var pbSeed quorumpb.GroupSeed
		if err := json.Unmarshal(v, &pbSeed); err != nil {
			return err
		}
		seeds[string(k)] = &pbSeed

		return nil
	})

	return seeds, err
}

// Get group list
func (cs *Storage) GetSubGroupItems(parentGroupId string) ([]*quorumpb.GroupItem, error) {
	var groupItems []*quorumpb.GroupItem
	key := s.GetGroupItemPrefix(parentGroupId)

	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		groupItem := &quorumpb.GroupItem{}
		err = proto.Unmarshal(v, groupItem)
		if err != nil {
			return err
		}

		groupItems = append(groupItems, groupItem)
		return nil
	})
	return groupItems, err
}

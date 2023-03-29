package chainstorage

import (
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) AddGroup(groupItem *quorumpb.GroupItem) error {
	//check if group exist
	key := s.GetGroupItemKey(groupItem.GroupId)
	exist, _ := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) UpdGroup(groupItem *quorumpb.GroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	key := s.GetGroupItemKey(groupItem.GroupId)
	//upd group to db
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) RmGroup(groupId string) error {
	//check if group exist
	key := s.GetGroupItemKey(groupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("Group Not Found")
	}

	//delete group
	return cs.dbmgr.GroupInfoDb.Delete([]byte(key))
}

func (cs *Storage) GetGroupInfo(groupId string) (*quorumpb.GroupItem, error) {
	//check if group exist
	key := s.GetGroupItemKey(groupId)
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

	//all group producer
	key = s.GetProducerPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group users
	key = s.GetUserPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group announced item
	key = s.GetAnnouncedPrefix(groupId, prefix...)
	keys = append(keys, key)

	//all group schema item
	key = s.GetSchemaPrefix(groupId, prefix...)
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

	return nil
}

func (cs *Storage) AddGroupV2(groupItem *quorumpb.NodeSDKGroupItem) error {
	//check if group exist
	key := s.GetGroupItemKey(groupItem.Group.GroupId)
	exist, _ := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

// Get Gorup Info
func (cs *Storage) GetGroupInfoV2(groupId string) (*quorumpb.NodeSDKGroupItem, error) {
	key := s.GetGroupItemKey(groupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Group Not Found")
	}

	groupInfoByte, err := cs.dbmgr.GroupInfoDb.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var groupInfo *quorumpb.NodeSDKGroupItem
	groupInfo = &quorumpb.NodeSDKGroupItem{}
	err = proto.Unmarshal(groupInfoByte, groupInfo)
	if err != nil {
		return nil, err
	}

	return groupInfo, nil
}

func (cs *Storage) GetAllGroupsV2() ([]*quorumpb.NodeSDKGroupItem, error) {
	var result []*quorumpb.NodeSDKGroupItem

	key := s.GetGroupItemKey("")
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

func (cs *Storage) SetGroupSeed(seed *quorumpb.GroupSeed) error {
	key := s.GetSeedKey(seed.GenesisBlock.GroupId)
	value, err := proto.Marshal(seed)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set(key, value)
}

func (cs *Storage) GetGroupSeed(groupID string) (*quorumpb.GroupSeed, error) {
	key := s.GetSeedKey(groupID)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist(key)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("Group seed not exist")
	}

	value, err := cs.dbmgr.GroupInfoDb.Get(key)
	if err != nil {
		return nil, err
	}

	var result quorumpb.GroupSeed
	if err := proto.Unmarshal(value, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (cs *Storage) UpdGroupV2(groupItem *quorumpb.NodeSDKGroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	key := s.GetGroupItemKey(groupItem.Group.GroupId)
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("Group is not existed")
	}

	//upd group to db
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

package chainstorage

import (
	"errors"
	"fmt"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) AddGroup(groupItem *quorumpb.GroupItem) error {
	//check if group exist
	key := s.GROUPITEM_PREFIX + "_" + groupItem.GroupId
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
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

	key := s.GROUPITEM_PREFIX + "_" + groupItem.GroupId
	//upd group to db
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) RmGroup(groupId string) error {
	//check if group exist
	key := s.GROUPITEM_PREFIX + "_" + groupId
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

func (cs *Storage) RemoveGroupData(item *quorumpb.GroupItem, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	var keys []string

	//remove all group POST
	key := nodeprefix + s.GRP_PREFIX + "_" + s.CNT_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group producer
	key = nodeprefix + s.PRD_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group users
	key = nodeprefix + s.USR_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group announced item
	key = nodeprefix + s.ANN_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group schema item
	key = nodeprefix + s.SMA_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group chain_config item
	key = nodeprefix + s.CHAIN_CONFIG_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group app_config item
	key = nodeprefix + s.APP_CONFIG_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//nonce prefix
	key = nodeprefix + s.NONCE_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//snapshot
	key = nodeprefix + s.SNAPSHOT_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//remove all
	for _, key_prefix := range keys {
		_, err := cs.dbmgr.Db.PrefixDelete([]byte(key_prefix))
		if err != nil {
			return err
		}
	}

	keys = nil
	//remove all cached block
	key = nodeprefix + s.BLK_PREFIX + "_"
	keys = append(keys, key)
	key = nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_"
	keys = append(keys, key)

	for _, key_prefix := range keys {
		_, err := cs.dbmgr.Db.PrefixCondDelete([]byte(key_prefix), func(k []byte, v []byte, err error) (bool, error) {
			if err != nil {
				return false, err
			}

			blockChunk := quorumpb.BlockDbChunk{}
			perr := proto.Unmarshal(v, &blockChunk)
			if perr != nil {
				return false, perr
			}

			if blockChunk.BlockItem.GroupId == item.GroupId {
				return true, nil
			}
			return false, nil
		})

		if err != nil {
			return err
		}
	}

	//remove all trx
	key = nodeprefix + s.TRX_PREFIX + "_"
	_, err := cs.dbmgr.Db.PrefixCondDelete([]byte(key), func(k []byte, v []byte, err error) (bool, error) {
		if err != nil {
			return false, err
		}

		trx := quorumpb.Trx{}
		perr := proto.Unmarshal(v, &trx)

		if perr != nil {
			return false, perr
		}

		if trx.GroupId == item.GroupId {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (cs *Storage) AddGroupV2(groupItem *quorumpb.NodeSDKGroupItem) error {
	//check if group exist
	key := s.GROUPITEM_PREFIX + "_" + groupItem.Group.GroupId
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
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

//Get Gorup Info
func (cs *Storage) GetGroupInfoV2(groupId string) (*quorumpb.NodeSDKGroupItem, error) {
	key := s.GROUPITEM_PREFIX + "_" + groupId
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

	key := s.GROUPITEM_PREFIX
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		var item *quorumpb.NodeSDKGroupItem
		item = &quorumpb.NodeSDKGroupItem{}
		err = proto.Unmarshal(v, item)
		if err != nil {
			return err
		}
		result = append(result, item)
		return nil
	})
	return result, err
}

func groupSeedKey(groupID string) []byte {
	return []byte(fmt.Sprintf("%s_%s", s.GROUPSEED_PREFIX, groupID))
}

func (cs *Storage) SetGroupSeed(seed *quorumpb.GroupSeed) error {
	key := groupSeedKey(seed.GenesisBlock.GroupId)
	value, err := proto.Marshal(seed)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set(key, value)
}

func (cs *Storage) GetGroupSeed(groupID string) (*quorumpb.GroupSeed, error) {
	key := groupSeedKey(groupID)
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

	key := s.GROUPITEM_PREFIX + "_" + groupItem.Group.GroupId
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

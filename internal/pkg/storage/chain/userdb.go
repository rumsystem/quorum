package chainstorage

import (
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateUserTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return cs.UpdateUser(trx.Data, prefix...)
}

func (cs *Storage) UpdateUser(data []byte, prefix ...string) (err error) {
	item := &quorumpb.UserItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	key := s.GetUserKey(item.GroupId, item.UserPubkey, prefix...)
	chaindb_log.Infof("update user with key %s", key)

	if item.Action == quorumpb.ActionType_ADD {
		chaindb_log.Infof("Add user")
		return cs.dbmgr.Db.Set([]byte(key), data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if group exist
		chaindb_log.Infof("Remove user")
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("User Not Found")
		}

		return cs.dbmgr.Db.Delete([]byte(key))
	} else {
		chaindb_log.Infof("unknow msgType")
		return errors.New("unknow msgType")
	}
}

func (cs *Storage) GetAllUserInBytes(groupId string, prefix ...string) ([][]byte, error) {
	key := s.GetUserPrefix(groupId, prefix...)
	var usersByteList [][]byte

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		usersByteList = append(usersByteList, v)
		return nil
	})

	return usersByteList, err
}

func (cs *Storage) GetAnnounceUsersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	key := s.GetAnnounceAsUserPrefix(groupId, prefix...)
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

func (cs *Storage) GetAnnouncedUser(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	key := s.GetAnnounceAsUserKey(groupId, pubkey, prefix...)
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var ann quorumpb.AnnounceItem
	err = proto.Unmarshal(value, &ann)
	if err != nil {
		return nil, err
	}

	return &ann, err
}

func (cs *Storage) IsUserAnnounced(groupId, userSignPubkey string, prefix ...string) (bool, error) {
	key := s.GetAnnounceAsUserKey(groupId, userSignPubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

// IsUser check if group user (announced) exist
func (cs *Storage) IsUser(groupId, userPubKey string, prefix ...string) (bool, error) {
	key := s.GetAnnounceAsUserKey(groupId, userPubKey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

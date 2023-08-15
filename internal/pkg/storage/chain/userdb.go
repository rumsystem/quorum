package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) GetUsers(groupId string, prefix ...string) ([]*quorumpb.UserItem, error) {
	var uList []*quorumpb.UserItem

	key := s.GetUserPrefix(groupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.UserItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		uList = append(uList, &item)
		return nil
	})

	return uList, err
}

func (cs *Storage) GetUser(groupId string, pubkey string, prefix ...string) (*quorumpb.UserItem, error) {
	key := s.GetUserKey(groupId, pubkey, prefix...)
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var user quorumpb.UserItem
	err = proto.Unmarshal(value, &user)
	if err != nil {
		return nil, err
	}

	return &user, err
}

func (cs *Storage) IsUser(groupId, userSignPubkey string, prefix ...string) (bool, error) {
	key := s.GetUserKey(groupId, userSignPubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

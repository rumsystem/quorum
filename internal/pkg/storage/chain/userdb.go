package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateGroupUser(trxId string, data []byte, prefix ...string) (err error) {

	/*
		item := &quorumpb.UpdGroupUserItem{}
		if err := proto.Unmarshal(data, item); err != nil {
			return err
		}

		if item.Action == quorumpb.ActionType_ADD {
			announceItemKey := s.GetAnnounceAsUserKey(item.GroupId, item.UserPubkey, prefix...)
			exist, err := cs.dbmgr.Db.IsExist([]byte(announceItemKey))
			if !exist {
				if err != nil {
					return err
				}
				return errors.New("announce item not found")
			}

			announcedItemBytes, err := cs.dbmgr.Db.Get([]byte(announceItemKey))
			if err != nil {
				return err
			}

			announcedItem := &quorumpb.AnnounceItem{}
			if err := proto.Unmarshal(announcedItemBytes, announcedItem); err != nil {
				return err
			}

			userItem := &quorumpb.UserItem{
				GroupId:       announcedItem.GroupId,
				UserPubkey:    announcedItem.Content.SignPubkey,
				EncryptPubkey: announcedItem.Content.EncryptPubkey,
				ProofTrxId:    trxId,
				TxCnt:         0,
				Memo:          announcedItem.Content.Memo,
			}

			data, err := proto.Marshal(userItem)
			if err != nil {
				return err
			}

			key := s.GetUserKey(item.GroupId, item.UserPubkey, prefix...)
			return cs.dbmgr.Db.Set([]byte(key), data)
		} else if item.Action == quorumpb.ActionType_REMOVE {

			userKey := s.GetUserKey(item.GroupId, item.UserPubkey, prefix...)
			exist, err := cs.dbmgr.Db.IsExist([]byte(userKey))
			if !exist {
				if err != nil {
					return err
				}
				return errors.New("user not found")
			}

			return cs.dbmgr.Db.Delete([]byte(userKey))
		} else {
			chaindb_log.Infof("unknow msgType")
			return errors.New("unknow msgType")
		}
	*/

	return nil
}

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

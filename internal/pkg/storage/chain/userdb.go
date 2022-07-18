package chainstorage

import (
	"errors"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateUserTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return cs.UpdateUser(trx.Data, prefix...)
}

func (cs *Storage) UpdateUser(data []byte, prefix ...string) (err error) {

	nodeprefix := utils.GetPrefix(prefix...)

	item := &quorumpb.UserItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.UserPubkey)
	if pk == "" {
		pk = item.UserPubkey
	}

	key := nodeprefix + s.USR_PREFIX + "_" + item.GroupId + "_" + pk
	chaindb_log.Infof("upd user with key %s", key)

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

func (cs *Storage) GetAllUserInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + s.USR_PREFIX + "_" + groupId + "_"
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

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String()
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
	nodeprefix := utils.GetPrefix(prefix...)

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(pubkey)
	if pk == "" {
		pk = pubkey
	}

	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + pk

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
	nodeprefix := utils.GetPrefix(prefix...)

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(userSignPubkey)
	if pk == "" {
		pk = userSignPubkey
	}

	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + pk
	return cs.dbmgr.Db.IsExist([]byte(key))
}

func (cs *Storage) IsUser(groupId, userPubKey string, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(userPubKey)
	if pk == "" {
		pk = userPubKey
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + pk

	//check if group user (announced) exist
	return cs.dbmgr.Db.IsExist([]byte(key))
}

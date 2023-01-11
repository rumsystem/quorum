package chainstorage

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type Storage struct {
	dbmgr *s.DbMgr
}

var chaindb_log = logging.Logger("chaindb")

func NewChainStorage(dbmgr *s.DbMgr) (storage *Storage) {
	if storage == nil {
		storage = &Storage{dbmgr}
	}
	return storage
}

func (cs *Storage) UpdateAnnounceResult(announcetype quorumpb.AnnounceType, groupId, signPubkey string, result bool, prefix ...string) error {
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(signPubkey)
	if pk == "" {
		pk = signPubkey
	}
	key := s.GetAnnouncedKey(groupId, announcetype.String(), pk, prefix...)
	pAnnounced := &quorumpb.AnnounceItem{}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		//patch for old keyformat
		key := s.GetAnnouncedKey(groupId, announcetype.String(), signPubkey, prefix...)
		value, err = cs.dbmgr.Db.Get([]byte(key))
		if err != nil {
			return err
		}
		//update to the new keyformat
		key = s.GetAnnouncedKey(groupId, announcetype.String(), pk, prefix...)
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
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		chaindb_log.Debugf(err.Error())
		return err
	}
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.SignPubkey)
	if pk == "" {
		pk = item.SignPubkey
	}
	key := s.GetAnnouncedKey(item.GroupId, item.Type.Enum().String(), pk, prefix...)
	err = cs.dbmgr.Db.Set([]byte(key), data)
	if err != nil {
		chaindb_log.Debugf("error %s", err.Error())
	}

	return err
}

func (cs *Storage) GetUsers(groupId string, prefix ...string) ([]*quorumpb.UserItem, error) {
	var pList []*quorumpb.UserItem
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
		pList = append(pList, &item)
		return nil
	})
	return pList, err
}

func (cs *Storage) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var pList []*quorumpb.ProducerItem
	key := s.GetProducerPrefix(groupId, prefix...)

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
	key := s.GetAnnounceAsProducerPrefix(groupId, prefix...)

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

	if err != nil {
		chaindb_log.Debugf("error %s", err.Error())
	}

	return aList, err
}

func (cs *Storage) AddPost(trx *quorumpb.Trx, prefix ...string) error {
	key := s.GetPostKey(trx.GroupId, fmt.Sprint(trx.TimeStamp), trx.TrxId, prefix...)
	chaindb_log.Debugf("Add POST with key %s", key)

	ctnItem := &quorumpb.PostItem{}

	ctnItem.TrxId = trx.TrxId
	ctnItem.SenderPubkey = trx.SenderPubkey
	ctnItem.Content = trx.Data
	ctnItem.TimeStamp = trx.TimeStamp
	ctnBytes, err := proto.Marshal(ctnItem)
	if err != nil {
		return err
	}

	return cs.dbmgr.Db.Set([]byte(key), ctnBytes)
}

// TBD
func (cs *Storage) SaveChainInfo(currEpoch, lastUpdate int64, groupId string, prefix ...string) error {
	return nil
}

// TBD
func (cs *Storage) GetChainInfo(groupId string, prefix ...string) (currEpoch int64, lastUpdate int64, err error) {
	return 0, 0, nil
}

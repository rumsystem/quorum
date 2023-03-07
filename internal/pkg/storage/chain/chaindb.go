package chainstorage

import (
	"encoding/binary"
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
func (cs *Storage) SaveChainInfo(currBlock, currEpoch uint64, lastUpdate int64, groupId string, prefix ...string) error {

	key := s.GetChainInfoEpoch(groupId, prefix...)
	chaindb_log.Debugf("Save ChainInfo, currEpoch <%d>", currEpoch)
	e := make([]byte, 8)
	binary.LittleEndian.PutUint64(e, currEpoch)
	cs.dbmgr.Db.Set([]byte(key), e)

	key = s.GetChainInfoBlock(groupId, prefix...)
	chaindb_log.Debugf("Save ChainInfo, currBlock <%d>", currBlock)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, currBlock)
	cs.dbmgr.Db.Set([]byte(key), b)

	key = s.GetChainInfoLastUpdate(groupId, prefix...)
	chaindb_log.Debugf("Save ChainInfo, LastUpdate <%d>", lastUpdate)
	l := make([]byte, 8)
	binary.LittleEndian.PutUint64(l, uint64(lastUpdate))
	cs.dbmgr.Db.Set([]byte(key), l)

	return nil
}

// TBD
func (cs *Storage) GetChainInfo(groupId string, prefix ...string) (currBlock, currEpoch uint64, lastUpdate int64, err error) {
	key := s.GetChainInfoEpoch(groupId, prefix...)
	e, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return 0, 0, 0, err
	}
	epoch := binary.LittleEndian.Uint64(e)
	chaindb_log.Debugf("Load ChainInfo, currEpoch <%d>", epoch)

	key = s.GetChainInfoBlock(groupId, prefix...)
	b, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return 0, 0, 0, err
	}

	block := binary.LittleEndian.Uint64(b)
	chaindb_log.Debugf("Load ChainInfo, currBlock <%d>", block)

	key = s.GetChainInfoLastUpdate(groupId, prefix...)
	l, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return 0, 0, 0, err
	}
	last := int64(binary.LittleEndian.Uint64(l))
	chaindb_log.Debugf("Load ChainInfo, LastUpdate <%d>", last)
	return block, epoch, last, nil
}

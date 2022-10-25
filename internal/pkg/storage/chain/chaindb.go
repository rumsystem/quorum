package chainstorage

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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
	nodeprefix := utils.GetPrefix(prefix...)

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(signPubkey)
	if pk == "" {
		pk = signPubkey
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + announcetype.String() + "_" + pk
	pAnnounced := &quorumpb.AnnounceItem{}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		//patch for old keyformat
		key = nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + announcetype.String() + "_" + signPubkey
		value, err = cs.dbmgr.Db.Get([]byte(key))
		if err != nil {
			return err
		}
		//update to the new keyformat
		key = nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + announcetype.String() + "_" + pk
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
		chaindb_log.Debugf(err.Error())
		return err
	}
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.SignPubkey)
	if pk == "" {
		pk = item.SignPubkey
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + item.GroupId + "_" + item.Type.Enum().String() + "_" + pk
	err = cs.dbmgr.Db.Set([]byte(key), data)
	if err != nil {
		chaindb_log.Debugf("error %s", err.Error())
	}

	return err
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

	if err != nil {
		chaindb_log.Debugf("error %s", err.Error())
	}

	return aList, err
}

func (cs *Storage) AddPost(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.GRP_PREFIX + "_" + s.CNT_PREFIX + "_" + trx.GroupId + "_" + fmt.Sprint(trx.TimeStamp) + "_" + trx.TrxId
	chaindb_log.Debugf("Add POST with key %s", key)

	ctnItem := &quorumpb.PostItem{}

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

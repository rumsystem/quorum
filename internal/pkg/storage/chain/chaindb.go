package chainstorage

import (
	"encoding/binary"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
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

func (cs *Storage) AddAnnounceItem(item *quorumpb.AnnounceItem, prefix ...string) (err error) {
	var key string
	if item.Content.Type == quorumpb.AnnounceType_AS_USER {
		key = s.GetAnnounceAsUserKey(item.GroupId, item.Content.SignPubkey, prefix...)
	} else if item.Content.Type == quorumpb.AnnounceType_AS_PRODUCER {
		key = s.GetAnnounceAsProducerKey(item.GroupId, item.Content.SignPubkey, prefix...)
	} else {
		return fmt.Errorf("unknown announce type %d", item.Content.Type)
	}

	data, err := proto.Marshal(item)
	if err != nil {
		chaindb_log.Debugf(err.Error())
		return err
	}

	err = cs.dbmgr.Db.Set([]byte(key), data)
	if err != nil {
		chaindb_log.Debugf(err.Error())
		return err
	}

	return nil
}

func (cs *Storage) UpdateAnnounce(data []byte, prefix ...string) (err error) {
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		chaindb_log.Debugf(err.Error())
		return err
	}

	var key string
	if item.Content.Type == quorumpb.AnnounceType_AS_USER {
		key = s.GetAnnounceAsUserKey(item.GroupId, item.Content.SignPubkey, prefix...)
	} else if item.Content.Type == quorumpb.AnnounceType_AS_PRODUCER {
		key = s.GetAnnounceAsProducerKey(item.GroupId, item.Content.SignPubkey, prefix...)
	} else {
		return fmt.Errorf("unknown announce type %d", item.Content.Type)
	}

	if item.Action == quorumpb.ActionType_ADD {
		//check if already exist
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if exist {
			if err != nil {
				return err
			}
			return fmt.Errorf("announce item already exist")
		}
		//add item to db
		err = cs.dbmgr.Db.Set([]byte(key), data)
		if err != nil {
			chaindb_log.Debugf("error %s", err.Error())
		}
	} else if item.Action == quorumpb.ActionType_REMOVE {
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return fmt.Errorf("announce item not exist")
		}
		//remove item from db
		err = cs.dbmgr.Db.Delete([]byte(key))
		if err != nil {
			chaindb_log.Debugf("error %s", err.Error())
		}
	} else {
		return fmt.Errorf("unknown action type %d", item.Action.Type())
	}

	return nil
}

func (cs *Storage) AddPost(trx *quorumpb.Trx, decodedData []byte, prefix ...string) error {
	key := s.GetPostKey(trx.GroupId, fmt.Sprint(trx.TimeStamp), trx.TrxId, prefix...)

	ctnItem := &quorumpb.PostItem{}

	ctnItem.TrxId = trx.TrxId
	ctnItem.SenderPubkey = trx.SenderPubkey
	ctnItem.Content = decodedData
	ctnItem.TimeStamp = trx.TimeStamp
	ctnBytes, err := proto.Marshal(ctnItem)
	if err != nil {
		return err
	}

	return cs.dbmgr.Db.Set([]byte(key), ctnBytes)
}

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

func (cs *Storage) UpdateChangeConsensusResult(groupId string, result *quorumpb.ChangeConsensusResultBundle, prefix ...string) error {
	key := s.GetChangeConsensusResultKey(groupId, result.Req.ReqId, prefix...)
	chaindb_log.Debugf("UpdateChangeConsensusResult key %s", key)
	data, err := proto.Marshal(result)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), data)
}

func (cs *Storage) GetAllChangeConsensusResult(groupId string, prefix ...string) ([]*quorumpb.ChangeConsensusResultBundle, error) {
	var rList []*quorumpb.ChangeConsensusResultBundle
	chaindb_log.Debugf("GetAllChangeConsensusResult called")

	key := s.GetChangeConsensusResultPrefix(groupId, prefix...)
	chaindb_log.Debugf("GetAllChangeConsensusResult key %s", key)

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.ChangeConsensusResultBundle{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		rList = append(rList, &item)
		return nil
	})

	return rList, err
}

func (cs *Storage) GetChangeConsensusResultByReqId(groupId, reqId string, prefix ...string) (*quorumpb.ChangeConsensusResultBundle, error) {
	key := s.GetChangeConsensusResultKey(groupId, reqId, prefix...)
	data, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	item := quorumpb.ChangeConsensusResultBundle{}
	perr := proto.Unmarshal(data, &item)
	if perr != nil {
		return nil, perr
	}
	return &item, nil
}

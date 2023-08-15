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
	//chaindb_log.Debugf("Save ChainInfo, currEpoch <%d>", currEpoch)
	e := make([]byte, 8)
	binary.LittleEndian.PutUint64(e, currEpoch)
	cs.dbmgr.Db.Set([]byte(key), e)

	key = s.GetChainInfoBlock(groupId, prefix...)
	//chaindb_log.Debugf("Save ChainInfo, currBlock <%d>", currBlock)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, currBlock)
	cs.dbmgr.Db.Set([]byte(key), b)

	key = s.GetChainInfoLastUpdate(groupId, prefix...)
	//chaindb_log.Debugf("Save ChainInfo, LastUpdate <%d>", lastUpdate)
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

func (cs *Storage) SaveGroupConsensusInfo(groupId string, info *quorumpb.ConsensusInfo, prefix ...string) error {
	key := s.GetGroupConsensusInfoKey(groupId, prefix...)
	chaindb_log.Debugf("Save GroupConsensusInfo, key <%s>", key)
	data, err := proto.Marshal(info)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), data)
}

func (cs *Storage) GetGroupConsensusInfo(groupId string, prefix ...string) (info *quorumpb.ConsensusInfo, err error) {
	key := s.GetGroupConsensusInfoKey(groupId, prefix...)
	chaindb_log.Debugf("Get GroupConsensusInfo, key <%s>", key)
	data, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	info = &quorumpb.ConsensusInfo{}
	err = proto.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

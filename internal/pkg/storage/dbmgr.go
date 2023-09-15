package storage

import (
	"errors"
	"sync"

	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var dbmgr_log = logging.Logger("dbmgr")

type DbMgr struct {
	GroupInfoDb QuorumStorage
	Db          QuorumStorage
	seq         sync.Map
	DataPath    string
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbmgr_log.Infof("GroupInfoDb closed")
	dbMgr.Db.Close()
	dbmgr_log.Infof("Db closed")
}

// should move to blockdb
// get block
func (dbMgr *DbMgr) GetBlock(groupId string, blockId uint64, cached bool, prefix ...string) (*quorumpb.Block, error) {
	var key string
	if cached {
		key = GetCachedBlockKey(groupId, blockId, prefix...)
	} else {
		key = GetBlockKey(groupId, blockId, prefix...)
	}

	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if err != nil {
		return nil, err
	}

	if !isExist {
		return nil, rumerrors.ErrBlockExist
	}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	block := quorumpb.Block{}
	err = proto.Unmarshal(value, &block)
	if err != nil {
		return nil, err
	}

	return &block, err
}

// save block chunk to cache datasource
func (dbMgr *DbMgr) SaveBlockToDSCache(block *quorumpb.Block, prefix ...string) error {
	// TODO PrevHash bytes to string
	key := GetDSCachedBlockPrefix(block.GroupId, block.BlockId, prefix...)
	dbmgr_log.Debugf("try save block to cache datasource with key <%s>", key)

	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}

	if isExist {
		return rumerrors.ErrBlockExist
	}

	value, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

// get block chunk from cache datasource
func (dbMgr *DbMgr) GetBlockFromDSCache(groupId string, blockId uint64, prefix ...string) (*quorumpb.Block, error) {
	key := GetDSCachedBlockPrefix(groupId, blockId, prefix...)
	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if err != nil {
		return nil, err
	}

	if !isExist {
		return nil, rumerrors.ErrBlockIDNotFound
	}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	block := quorumpb.Block{}
	err = proto.Unmarshal(value, &block)
	if err != nil {
		return nil, err
	}

	return &block, err
}

// save block chunk
func (dbMgr *DbMgr) SaveBlock(block *quorumpb.Block, cached bool, prefix ...string) error {
	var key string
	if cached {
		key = GetCachedBlockKey(block.GroupId, block.BlockId, prefix...)
	} else {
		key = GetBlockKey(block.GroupId, block.BlockId, prefix...)
	}
	dbmgr_log.Debugf("try save block with key <%s>", key)

	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}

	if isExist {
		return rumerrors.ErrBlockExist
	}

	value, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) RmBlock(groupId string, blockId uint64, cached bool, prefix ...string) error {
	var key string
	if cached {
		key = GetCachedBlockKey(groupId, blockId, prefix...)
	} else {
		key = GetBlockKey(groupId, blockId, prefix...)
	}
	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}

	if !isExist {
		return errors.New("block not exist")
	}

	return dbMgr.Db.Delete([]byte(key))
}

func (dbMgr *DbMgr) IsBlockExist(groupId string, blockId uint64, cached bool, prefix ...string) (bool, error) {
	var key string
	if cached {
		key = GetCachedBlockKey(groupId, blockId, prefix...)
	} else {
		key = GetBlockKey(groupId, blockId, prefix...)
	}
	return dbMgr.Db.IsExist([]byte(key))
}

// get next nonce
func (dbMgr *DbMgr) GetNextConsensusNonce(groupId string, prefix ...string) (uint64, error) {
	key := GetConsensusNonceKey(groupId, prefix...)
	nonceseq, succ := dbMgr.seq.Load(key)
	if !succ {
		newseq, err := dbMgr.Db.GetSequence([]byte(key), 1)
		if err != nil {
			return 0, err
		}
		dbMgr.seq.Store(key, newseq)
		return newseq.Next()
	} else {
		return nonceseq.(Sequence).Next()
	}
}

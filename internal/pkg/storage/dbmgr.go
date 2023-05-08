package storage

import (
	"errors"
	"strconv"
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
	Auth        QuorumStorage
	seq         sync.Map
	DataPath    string
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbMgr.Db.Close()
	//dbMgr.Auth.Close()
	dbmgr_log.Infof("ChainCtx Db closed")
}

func (dbMgr *DbMgr) TryMigration(nodeDataVer int) {
	//no need run migration for the first version
}

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

// Get group list
func (dbMgr *DbMgr) GetGroupsBytes() ([][]byte, error) {
	var groupItemList [][]byte
	key := GetGroupItemPrefix()

	err := dbMgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		groupItemList = append(groupItemList, v)
		return nil
	})
	return groupItemList, err
}

func (dbMgr *DbMgr) GetAllAnnounceInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	key := GetAnnouncedPrefix(groupId, Prefix...)
	var announceByteList [][]byte

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		announceByteList = append(announceByteList, v)
		return nil
	})

	return announceByteList, err
}

func (dbMgr *DbMgr) GetAppConfigItemInt(itemKey string, groupId string, prefix ...string) (int, error) {
	key := GetAppConfigKey(groupId, itemKey, prefix...)
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return -1, err
	}

	var config quorumpb.AppConfigItem
	err = proto.Unmarshal(value, &config)
	if err != nil {
		return -1, err
	}

	result, err := strconv.Atoi(config.Value)
	return result, err
}

func (dbMgr *DbMgr) GetAppConfigItemBool(itemKey string, groupId string, prefix ...string) (bool, error) {
	key := GetAppConfigKey(groupId, itemKey, prefix...)
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return false, err
	}

	var config quorumpb.AppConfigItem
	err = proto.Unmarshal(value, &config)
	if err != nil {
		return false, err
	}

	result, err := strconv.ParseBool(config.Value)
	return result, err
}

func (dbMgr *DbMgr) GetAppConfigItemString(itemKey string, groupId string, prefix ...string) (string, error) {
	key := GetAppConfigKey(groupId, itemKey, prefix...)
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return "", err
	}

	var config quorumpb.AppConfigItem
	err = proto.Unmarshal(value, &config)
	if err != nil {
		return "", err
	}

	return config.Value, err
}

func (dbMgr *DbMgr) GetAnnouncedEncryptKeys(groupId string, prefix ...string) (pubkeylist []string, err error) {
	keys := []string{}
	return keys, nil
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

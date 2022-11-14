package storage

import (
	"errors"
	"strconv"
	"strings"
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
	if nodeDataVer == 0 { //try migration 0 (Upgrade the GroupItem)
		dbmgr_log.Infof("db migration v0")
		groupItemsBytes, err := dbMgr.GetGroupsBytes()
		if err == nil {
			for _, b := range groupItemsBytes {
				item := &quorumpb.GroupItem{}
				proto.Unmarshal(b, item)
				if item.CipherKey == "" {
					itemv0 := &quorumpb.GroupItemV0{}
					proto.Unmarshal(b, itemv0)
					if itemv0.CipherKey != "" { //ok
						item.LastUpdate = itemv0.LastUpdate
						item.Epoch = itemv0.HighestHeight
						//item.HighestBlockId = itemv0.HighestBlockId
						item.GenesisBlock = itemv0.GenesisBlock
						item.EncryptType = itemv0.EncryptType
						item.ConsenseType = itemv0.ConsenseType
						item.CipherKey = itemv0.CipherKey
						item.AppKey = itemv0.AppKey
						//add group to db
						value, err := proto.Marshal(item)
						if err == nil {
							dbMgr.GroupInfoDb.Set([]byte(item.GroupId), value)
							dbmgr_log.Infof("db migration v0 for group %s", item.GroupId)
						}
					}
				}
			}
		}
	}

	if nodeDataVer == 1 { //try migration 1 (Upgrade the GroupInfodb key with GROUPITEM_PREFIX prefix)
		err := dbMgr.GroupInfoDb.Foreach(func(k []byte, v []byte, err error) error {
			key := string(k)
			if len(key) == 36 && strings.Contains(key, "_") == false {
				newkey := GetGroupItemKey(key)
				err = dbMgr.GroupInfoDb.Set([]byte(newkey), v)
				if err == nil {
					dbmgr_log.Infof("db migration v1 for group %s", key)
					return dbMgr.GroupInfoDb.Delete([]byte(key))
				} else {
					return err
				}
			}
			return nil
		})

		if err != nil {
			dbmgr_log.Errorf("db migration v1 for groupinfodb err %s", err)
		}
	}
}

// get block
func (dbMgr *DbMgr) GetBlock(groupId string, epoch int64, cached bool, prefix ...string) (*quorumpb.Block, error) {
	var key string
	if cached {
		key = GetCachedBlockKey(groupId, epoch, prefix...)
	} else {
		key = GetBlockKey(groupId, epoch, prefix...)
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
	dbmgr_log.Debug("SaveBlock called")
	var key string
	if cached {
		key = GetCachedBlockKey(block.GroupId, block.Epoch, prefix...)
	} else {
		key = GetBlockKey(block.GroupId, block.Epoch, prefix...)
	}
	dbmgr_log.Debugf("KEY %s", key)

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

func (dbMgr *DbMgr) RmBlock(groupId string, epoch int64, cached bool, prefix ...string) error {
	var key string
	if cached {
		key = GetCachedBlockKey(groupId, epoch, prefix...)
	} else {
		key = GetBlockKey(groupId, epoch, prefix...)
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

func (dbMgr *DbMgr) IsBlockExist(groupId string, epoch int64, cached bool, prefix ...string) (bool, error) {
	var key string
	if cached {
		key = GetCachedBlockKey(groupId, epoch, prefix...)
	} else {
		key = GetBlockKey(groupId, epoch, prefix...)
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
func (dbMgr *DbMgr) GetNextNouce(groupId string, prefix ...string) (uint64, error) {
	key := GetNonceKey(groupId, prefix...)
	nonceseq, succ := dbMgr.seq.Load(key)
	if succ == false {
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

//func (dbMgr *DbMgr) GetGrpCtnt(groupId string, ctntype string, prefix ...string) ([]*quorumpb.PostItem, error) {
//	var ctnList []*quorumpb.PostItem
//	nodeprefix := utils.GetPrefix(prefix...)
//	pre := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + groupId + "_"
//	err := dbMgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
//		if err != nil {
//			return err
//		}
//
//		item := quorumpb.PostItem{}
//		perr := proto.Unmarshal(v, &item)
//		if perr != nil {
//			return perr
//		}
//		ctnList = append(ctnList, &item)
//		return nil
//	})
//
//	return ctnList, err
//}

//func (dbMgr *DbMgr) GetTrxContent(trxId string, prefix ...string) (*quorumpb.Trx, error) {
//	nodeprefix := utils.GetPrefix(prefix...)
//	var trx quorumpb.Trx
//	key := nodeprefix + TRX_PREFIX + "_" + trxId
//	err := dbMgr.Db.View(func(txn *badger.Txn) error {
//		item, err := txn.Get([]byte(key))
//		if err != nil {
//			return err
//		}
//
//		trxBytes, err := item.ValueCopy(nil)
//		if err != nil {
//			return err
//		}
//
//		err = proto.Unmarshal(trxBytes, &trx)
//		if err != nil {
//			return err
//		}
//
//		return nil
//	})
//
//	return &trx, err
//}

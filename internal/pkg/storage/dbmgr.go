package storage

import (
	"errors"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
	"strconv"
	"strings"
	"sync"
)

var dbmgr_log = logging.Logger("dbmgr")

const TRX_PREFIX string = "trx"                //trx
const BLK_PREFIX string = "blk"                //block
const GRP_PREFIX string = "grp"                //group
const CNT_PREFIX string = "cnt"                //content
const PRD_PREFIX string = "prd"                //producer
const USR_PREFIX string = "usr"                //user
const ANN_PREFIX string = "ann"                //announce
const SMA_PREFIX string = "sma"                //schema
const CHD_PREFIX string = "chd"                //cached
const APP_CONFIG_PREFIX string = "app_conf"    //group configuration
const CHAIN_CONFIG_PREFIX string = "chn_conf"  //chain configuration
const TRX_AUTH_TYPE_PREFIX string = "trx_auth" //trx auth type
const ALLW_LIST_PREFIX string = "alw_list"     //allow list
const DENY_LIST_PREFIX string = "dny_list"     //deny list
const NONCE_PREFIX string = "nonce"            //group trx nonce
const SNAPSHOT_PREFIX string = "snapshot"      //group snapshot

//groupinfo db
const GROUPITEM_PREFIX string = "grpitem" //relay
const RELAY_PREFIX string = "rly"         //relay

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
	dbmgr_log.Infof("ChainCtx Db closed")
}

func (dbMgr *DbMgr) TryMigration(nodeDataVer int) {
	if nodeDataVer == 0 { //try migration 0 (Upgrade the GroupItem)
		dbmgr_log.Infof("db migration v0")
		groupItemsBytes, err := dbMgr.GetGroupsBytes()
		if err == nil {
			for _, b := range groupItemsBytes {
				var item *quorumpb.GroupItem
				item = &quorumpb.GroupItem{}
				proto.Unmarshal(b, item)
				if item.CipherKey == "" {
					itemv0 := &quorumpb.GroupItemV0{}
					proto.Unmarshal(b, itemv0)
					if itemv0.CipherKey != "" { //ok
						item.LastUpdate = itemv0.LastUpdate
						item.HighestHeight = itemv0.HighestHeight
						item.HighestBlockId = itemv0.HighestBlockId
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
				newkey := GROUPITEM_PREFIX + "_" + key
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

//get block chunk
func (dbMgr *DbMgr) GetBlockChunk(blockId string, cached bool, prefix ...string) (*quorumpb.BlockDbChunk, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + blockId
	}

	pChunk := quorumpb.BlockDbChunk{}
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &pChunk)
	if err != nil {
		return nil, err
	}

	return &pChunk, err
}

//save block chunk
func (dbMgr *DbMgr) SaveBlockChunk(chunk *quorumpb.BlockDbChunk, cached bool, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + chunk.BlockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + chunk.BlockId
	}

	value, err := proto.Marshal(chunk)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) GetGrpCtnt(groupId string, ctntype string, prefix ...string) ([]*quorumpb.PostItem, error) {
	var ctnList []*quorumpb.PostItem
	nodeprefix := utils.GetPrefix(prefix...)
	pre := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + groupId + "_"
	err := dbMgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		item := quorumpb.PostItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		ctnList = append(ctnList, &item)
		return nil
	})

	return ctnList, err
}

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

//Get group list
func (dbMgr *DbMgr) GetGroupsBytes() ([][]byte, error) {
	var groupItemList [][]byte
	key := GROUPITEM_PREFIX + "_"

	err := dbMgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		groupItemList = append(groupItemList, v)
		return nil
	})
	return groupItemList, err
}

func (dbMgr *DbMgr) UpdateProducerTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return dbMgr.UpdateProducer(trx.Data, prefix...)
}

func (dbMgr *DbMgr) UpdateProducer(data []byte, prefix ...string) (err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	item := &quorumpb.ProducerItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	key := nodeprefix + PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey

	dbmgr_log.Infof("upd producer with key %s", key)

	if item.Action == quorumpb.ActionType_ADD {
		dbmgr_log.Infof("Add producer")
		return dbMgr.Db.Set([]byte(key), data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if group exist
		dbmgr_log.Infof("Remove producer")
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Producer Not Found")
		}

		return dbMgr.Db.Delete([]byte(key))
	} else {
		dbmgr_log.Infof("Remove producer")
		return errors.New("unknow msgType")
	}
}

func (dbMgr *DbMgr) UpdateUserTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return dbMgr.UpdateUser(trx.Data, prefix...)
}

func (dbMgr *DbMgr) UpdateUser(data []byte, prefix ...string) (err error) {

	nodeprefix := utils.GetPrefix(prefix...)

	item := &quorumpb.UserItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	key := nodeprefix + USR_PREFIX + "_" + item.GroupId + "_" + item.UserPubkey
	dbmgr_log.Infof("upd user with key %s", key)

	if item.Action == quorumpb.ActionType_ADD {
		dbmgr_log.Infof("Add user")
		return dbMgr.Db.Set([]byte(key), data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if group exist
		dbmgr_log.Infof("Remove user")
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("User Not Found")
		}

		return dbMgr.Db.Delete([]byte(key))
	} else {
		dbmgr_log.Infof("unknow msgType")
		return errors.New("unknow msgType")
	}
}

func (dbMgr *DbMgr) UpdateAppConfigTrx(trx *quorumpb.Trx, Prefix ...string) (err error) {
	return dbMgr.UpdateAppConfig(trx.Data, Prefix...)
}

func (dbMgr *DbMgr) UpdateAppConfig(data []byte, Prefix ...string) (err error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	item := &quorumpb.AppConfigItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + item.GroupId + "_" + item.Name

	if item.Action == quorumpb.ActionType_ADD {
		dbmgr_log.Infof("Add AppConfig item")
		return dbMgr.Db.Set([]byte(key), data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		dbmgr_log.Infof("Remove AppConfig item")
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("AppConfig key not Found")
		}

		return dbMgr.Db.Delete([]byte(key))
	} else {
		return errors.New("Unknown ACTION")
	}
}

// name, type
func (dbMgr *DbMgr) GetAppConfigKey(groupId string, prefix ...string) ([]string, []string, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + groupId

	var itemName []string
	var itemType []string

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.AppConfigItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		itemName = append(itemName, item.Name)
		itemType = append(itemType, item.Type.String())
		return nil
	})
	return itemName, itemType, err
}

func (dbMgr *DbMgr) GetAppConfigItem(itemKey string, groupId string, Prefix ...string) (*quorumpb.AppConfigItem, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + groupId + "_" + itemKey

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var config quorumpb.AppConfigItem
	err = proto.Unmarshal(value, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (dbMgr *DbMgr) GetAllAppConfigInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + groupId + "_"
	var appConfigByteList [][]byte

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		appConfigByteList = append(appConfigByteList, v)
		return nil
	})

	return appConfigByteList, err
}

func (dbMgr *DbMgr) GetAllChainConfigInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId + "_"
	var chainConfigByteList [][]byte

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		chainConfigByteList = append(chainConfigByteList, v)
		return nil
	})

	return chainConfigByteList, err
}

func (dbMgr *DbMgr) GetAllAnnounceInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_"
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

func (dbMgr *DbMgr) GetAllProducerInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId + "_"
	var producerByteList [][]byte

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		producerByteList = append(producerByteList, v)
		return nil
	})

	return producerByteList, err
}

func (dbMgr *DbMgr) GetAllUserInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + USR_PREFIX + "_" + groupId + "_"
	var usersByteList [][]byte

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		usersByteList = append(usersByteList, v)
		return nil
	})

	return usersByteList, err
}

func (dbMgr *DbMgr) GetAppConfigItemInt(itemKey string, groupId string, Prefix ...string) (int, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + groupId + "_" + itemKey

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

func (dbMgr *DbMgr) GetAppConfigItemBool(itemKey string, groupId string, Prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + groupId + "_" + itemKey

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

func (dbMgr *DbMgr) GetAppConfigItemString(itemKey string, groupId string, Prefix ...string) (string, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + APP_CONFIG_PREFIX + "_" + groupId + "_" + itemKey

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

func (dbMgr *DbMgr) AddProducer(item *quorumpb.ProducerItem, prefix ...string) error {

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey
	dbmgr_log.Infof("Add Producer with key %s", key)

	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), pbyte)
}

func (dbMgr *DbMgr) AddProducedBlockCount(groupId, producerPubkey string, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId + "_" + producerPubkey
	var pProducer *quorumpb.ProducerItem
	pProducer = &quorumpb.ProducerItem{}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(value, pProducer)
	if err != nil {
		return err
	}

	pProducer.BlockProduced += 1

	value, err = proto.Marshal(pProducer)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) GetAnnounceUsersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String()
	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
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

func (dbMgr *DbMgr) GetAnnouncedProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + pubkey

	value, err := dbMgr.Db.Get([]byte(key))
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

func (dbMgr *DbMgr) GetAnnouncedUser(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + pubkey

	value, err := dbMgr.Db.Get([]byte(key))
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

func (dbMgr *DbMgr) IsProducerAnnounced(groupId, producerSignPubkey string, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + producerSignPubkey
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) IsUserAnnounced(groupId, userSignPubkey string, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + userSignPubkey
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) IsUser(groupId, userPubKey string, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + userPubKey

	//check if group user (announced) exist
	return dbMgr.Db.IsExist([]byte(key))
}

//get next nonce
func (dbMgr *DbMgr) GetNextNouce(groupId string, prefix ...string) (uint64, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + NONCE_PREFIX + "_" + groupId

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

/*
   //test only, show db contents
   err = dbMgr.TrxDb.View(func(txn *badger.Txn) error {
           opts := badger.DefaultIteratorOptions
           opts.PrefetchSize = 10
           it := txn.NewIterator(opts)
           defer it.Close()
           for it.Rewind(); it.Valid(); it.Next() {
                   item := it.Item()
                   k := item.Key()
                   err := item.Value(func(v []byte) error {
                           fmt.Printf("key=%s, value=%s\n", k, v)
                           return nil
                   })
                   if err != nil {
                           return err
                   }
           }
           return nil
   })*/

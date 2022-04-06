package storage

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
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

type TrxStorageType uint

const (
	Chain TrxStorageType = iota
	Cache
)

type DbMgr struct {
	GroupInfoDb QuorumStorage
	Db          QuorumStorage
	Auth        QuorumStorage
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
}

//save trx
func (dbMgr *DbMgr) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trx.TrxId + "_" + fmt.Sprint(trx.Nonce)
	value, err := proto.Marshal(trx)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

//UNUSED
//rm Trx
func (dbMgr *DbMgr) RmTrx(trxId string, nonce int64, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId + "_" + fmt.Sprint(nonce)
	return dbMgr.Db.Delete([]byte(key))
}

//Get Trx
func (dbMgr *DbMgr) GetTrx(trxId string, storagetype TrxStorageType, prefix ...string) (t *quorumpb.Trx, n []int64, err error) {
	nodeprefix := getPrefix(prefix...)
	var trx quorumpb.Trx
	var nonces []int64

	var key string
	if storagetype == Chain {
		key = nodeprefix + TRX_PREFIX + "_" + trxId
		err = dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			perr := proto.Unmarshal(v, &trx)
			if perr != nil {
				return perr
			}
			nonces = append(nonces, trx.Nonce)
			return nil
		})
		trx.StorageType = quorumpb.TrxStroageType_CHAIN
	} else if storagetype == Cache {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX
		err = dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			chunk := quorumpb.BlockDbChunk{}
			perr := proto.Unmarshal(v, &chunk)
			if perr != nil {
				return perr
			}
			if chunk.BlockItem != nil && chunk.BlockItem.Trxs != nil {
				for _, blocktrx := range chunk.BlockItem.Trxs {
					if blocktrx.TrxId == trxId {
						nonces = append(nonces, blocktrx.Nonce)

						clonedtrxbuff, _ := proto.Marshal(blocktrx)
						perr = proto.Unmarshal(clonedtrxbuff, &trx)
						if perr != nil {
							return perr
						}
						trx.StorageType = quorumpb.TrxStroageType_CACHE
						return nil
					}
				}
			}

			return nil
		})

	}

	return &trx, nonces, err
}

func (dbMgr *DbMgr) UpdTrx(trx *quorumpb.Trx, prefix ...string) error {
	return dbMgr.AddTrx(trx, prefix...)
}

func (dbMgr *DbMgr) IsTrxExist(trxId string, nonce int64, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId + "_" + fmt.Sprint(nonce)

	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) AddGensisBlock(gensisBlock *quorumpb.Block, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + BLK_PREFIX + "_" + gensisBlock.BlockId

	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}
	if isExist {
		dbmgr_log.Debugf("Genesis block <%s> exist, do nothing", gensisBlock.BlockId)
		return nil
	}

	chunk := quorumpb.BlockDbChunk{}
	chunk.BlockId = gensisBlock.BlockId
	chunk.BlockItem = gensisBlock
	chunk.ParentBlockId = ""
	chunk.Height = 0

	value, err := proto.Marshal(&chunk)
	if err != nil {
		return err
	}

	return dbMgr.Db.Set([]byte(key), value)
}

//check if block existed
func (dbMgr *DbMgr) IsBlockExist(blockId string, cached bool, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + blockId
	}

	return dbMgr.Db.IsExist([]byte(key))
}

//check if parent block existed
func (dbMgr *DbMgr) IsParentExist(parentBlockId string, cached bool, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	var pKey string
	if cached {
		pKey = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + parentBlockId
	} else {
		pKey = nodeprefix + BLK_PREFIX + "_" + parentBlockId
	}

	return dbMgr.Db.IsExist([]byte(pKey))
}

//add block
func (dbMgr *DbMgr) AddBlock(newBlock *quorumpb.Block, cached bool, prefix ...string) error {
	isSaved, err := dbMgr.IsBlockExist(newBlock.BlockId, cached, prefix...)
	if err != nil {
		return err
	}

	if isSaved {
		dbmgr_log.Debugf("Block <%s> already saved, ignore", newBlock.BlockId)
		return nil
	}

	//create new chunk
	var chunk *quorumpb.BlockDbChunk
	chunk = &quorumpb.BlockDbChunk{}
	chunk.BlockId = newBlock.BlockId
	chunk.BlockItem = newBlock

	if cached {
		chunk.Height = -1        //Set height of cached chunk to -1
		chunk.ParentBlockId = "" //Set parent of cached chund to empty ""
	} else {
		//try get parent chunk
		pChunk, err := dbMgr.getBlockChunk(newBlock.PrevBlockId, cached, prefix...)
		if err != nil {
			return err
		}

		//update parent chunk
		pChunk.SubBlockId = append(pChunk.SubBlockId, chunk.BlockId)
		err = dbMgr.saveBlockChunk(pChunk, cached, prefix...)
		if err != nil {
			return err
		}

		chunk.Height = pChunk.Height + 1     //increase height
		chunk.ParentBlockId = pChunk.BlockId //point to parent
	}

	//save chunk
	return dbMgr.saveBlockChunk(chunk, cached, prefix...)
}

//remove block
func (dbMgr *DbMgr) RmBlock(blockId string, cached bool, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + blockId
	}

	return dbMgr.Db.Delete([]byte(key))
}

//get block by block_id
func (dbMgr *DbMgr) GetBlock(blockId string, cached bool, prefix ...string) (*quorumpb.Block, error) {
	pChunk, err := dbMgr.getBlockChunk(blockId, cached, prefix...)
	if err != nil {
		return nil, err
	}
	return pChunk.BlockItem, nil
}

func (dbMgr *DbMgr) GatherBlocksFromCache(newBlock *quorumpb.Block, cached bool, prefix ...string) ([]*quorumpb.Block, error) {
	nodeprefix := getPrefix(prefix...)
	var blocks []*quorumpb.Block
	blocks = append(blocks, newBlock)
	pointer1 := 0 //point to head
	pointer2 := 0 //point to tail

	pre := nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_"

	for {
		err := dbMgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			chunk := quorumpb.BlockDbChunk{}
			perr := proto.Unmarshal(v, &chunk)
			if perr != nil {
				return perr
			}
			if chunk.BlockItem.PrevBlockId == blocks[pointer1].BlockId {
				blocks = append(blocks, chunk.BlockItem)
				pointer2++
			}

			return nil
		})

		if err != nil {
			return blocks, err
		}

		if pointer1 == pointer2 {
			break
		}

		pointer1++
	}

	return blocks, nil
}

func (dbMgr *DbMgr) GetBlockHeight(blockId string, prefix ...string) (int64, error) {
	pChunk, err := dbMgr.getBlockChunk(blockId, false, prefix...)
	if err != nil {
		return -1, err
	}
	return pChunk.Height, nil
}

func (dbMgr *DbMgr) GetSubBlock(blockId string, prefix ...string) ([]*quorumpb.Block, error) {
	var result []*quorumpb.Block
	chunk, err := dbMgr.getBlockChunk(blockId, false, prefix...)
	if err != nil {
		return nil, err
	}

	for _, subChunkId := range chunk.SubBlockId {
		subChunk, err := dbMgr.getBlockChunk(subChunkId, false, prefix...)
		if err != nil {
			return nil, err
		}
		result = append(result, subChunk.BlockItem)
	}

	return result, nil
}

func (dbMgr *DbMgr) GetParentBlock(blockId string, prefix ...string) (*quorumpb.Block, error) {
	chunk, err := dbMgr.getBlockChunk(blockId, false, prefix...)
	if err != nil {
		return nil, err
	}

	parentChunk, err := dbMgr.getBlockChunk(chunk.ParentBlockId, false, prefix...)
	return parentChunk.BlockItem, err
}

//get block chunk
func (dbMgr *DbMgr) getBlockChunk(blockId string, cached bool, prefix ...string) (*quorumpb.BlockDbChunk, error) {
	nodeprefix := getPrefix(prefix...)
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
func (dbMgr *DbMgr) saveBlockChunk(chunk *quorumpb.BlockDbChunk, cached bool, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
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

func (dbMgr *DbMgr) AddGroup(groupItem *quorumpb.GroupItem) error {
	//check if group exist
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(groupItem.GroupId))
	if exist {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return dbMgr.GroupInfoDb.Set([]byte(groupItem.GroupId), value)
}

func (dbMgr *DbMgr) UpdGroup(groupItem *quorumpb.GroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	//upd group to db
	return dbMgr.GroupInfoDb.Set([]byte(groupItem.GroupId), value)
}

func (dbMgr *DbMgr) RmGroup(item *quorumpb.GroupItem) error {
	//check if group exist
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(item.GroupId))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("Group Not Found")
	}

	//delete group
	return dbMgr.GroupInfoDb.Delete([]byte(item.GroupId))
}

func (dbMgr *DbMgr) RemoveGroupData(item *quorumpb.GroupItem, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	var keys []string

	//remove all group POST
	key := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group producer
	key = nodeprefix + PRD_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group users
	key = nodeprefix + USR_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group announced item
	key = nodeprefix + ANN_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group schema item
	key = nodeprefix + SMA_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group chain_config item
	key = nodeprefix + CHAIN_CONFIG_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group app_config item
	key = nodeprefix + APP_CONFIG_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//nonce prefix
	key = nodeprefix + NONCE_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//snapshot
	key = nodeprefix + SNAPSHOT_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//remove all
	for _, key_prefix := range keys {
		err := dbMgr.Db.PrefixDelete([]byte(key_prefix))
		if err != nil {
			return err
		}
	}

	keys = nil
	//remove all cached block
	key = nodeprefix + BLK_PREFIX + "_"
	keys = append(keys, key)
	key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_"
	keys = append(keys, key)

	for _, key_prefix := range keys {
		err := dbMgr.Db.PrefixCondDelete([]byte(key_prefix), func(k []byte, v []byte, err error) (bool, error) {
			if err != nil {
				return false, err
			}

			blockChunk := quorumpb.BlockDbChunk{}
			perr := proto.Unmarshal(v, &blockChunk)
			if perr != nil {
				return false, perr
			}

			if blockChunk.BlockItem.GroupId == item.GroupId {
				return true, nil
			}
			return false, nil
		})

		if err != nil {
			return err
		}
	}

	//remove all trx
	key = nodeprefix + TRX_PREFIX + "_"
	err := dbMgr.Db.PrefixCondDelete([]byte(key), func(k []byte, v []byte, err error) (bool, error) {
		if err != nil {
			return false, err
		}

		trx := quorumpb.Trx{}
		perr := proto.Unmarshal(v, &trx)

		if perr != nil {
			return false, perr
		}

		if trx.GroupId == item.GroupId {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		return err
	}

	return nil
}

//Get group list
func (dbMgr *DbMgr) GetGroupsBytes() ([][]byte, error) {
	var groupItemList [][]byte
	err := dbMgr.GroupInfoDb.Foreach(func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		groupItemList = append(groupItemList, v)
		return nil
	})
	return groupItemList, err
}

func (dbMgr *DbMgr) AddPost(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + trx.GroupId + "_" + fmt.Sprint(trx.TimeStamp) + "_" + trx.TrxId
	dbmgr_log.Infof("Add POST with key %s", key)

	var ctnItem *quorumpb.PostItem
	ctnItem = &quorumpb.PostItem{}

	ctnItem.TrxId = trx.TrxId
	ctnItem.PublisherPubkey = trx.SenderPubkey
	ctnItem.Content = trx.Data
	ctnItem.TimeStamp = trx.TimeStamp
	ctnBytes, err := proto.Marshal(ctnItem)
	if err != nil {
		return err
	}

	return dbMgr.Db.Set([]byte(key), ctnBytes)
}

func (dbMgr *DbMgr) GetGrpCtnt(groupId string, ctntype string, prefix ...string) ([]*quorumpb.PostItem, error) {
	var ctnList []*quorumpb.PostItem
	nodeprefix := getPrefix(prefix...)
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
//	nodeprefix := getPrefix(prefix...)
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

func (dbMgr *DbMgr) UpdateChainConfigTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return dbMgr.UpdateChainConfig(trx.Data, prefix)
}

func (dbMgr *DbMgr) UpdateChainConfig(data []byte, prefix []string) (err error) {
	dbmgr_log.Infof("UpdateChainConfig called")
	nodeprefix := getPrefix(prefix...)
	item := &quorumpb.ChainConfigItem{}

	if err := proto.Unmarshal(data, item); err != nil {
		dbmgr_log.Infof(err.Error())
		return err
	}

	if item.Type == quorumpb.ChainConfigType_SET_TRX_AUTH_MODE {
		authModeItem := &quorumpb.SetTrxAuthModeItem{}
		if err := proto.Unmarshal(item.Data, authModeItem); err != nil {
			dbmgr_log.Infof(err.Error())
			return err
		}

		key := nodeprefix + CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + TRX_AUTH_TYPE_PREFIX + "_" + authModeItem.Type.String()
		return dbMgr.Db.Set([]byte(key), data)
	} else if item.Type == quorumpb.ChainConfigType_UPD_ALW_LIST ||
		item.Type == quorumpb.ChainConfigType_UPD_DNY_LIST {
		ruleListItem := &quorumpb.ChainSendTrxRuleListItem{}
		if err := proto.Unmarshal(item.Data, ruleListItem); err != nil {
			return err
		}

		var key string
		if item.Type == quorumpb.ChainConfigType_UPD_ALW_LIST {
			key = nodeprefix + CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + ALLW_LIST_PREFIX + "_" + ruleListItem.Pubkey
		} else {
			key = nodeprefix + CHAIN_CONFIG_PREFIX + "_" + item.GroupId + "_" + DENY_LIST_PREFIX + "_" + ruleListItem.Pubkey
		}

		dbmgr_log.Infof("key %s", key)

		if ruleListItem.Action == quorumpb.ActionType_ADD {
			return dbMgr.Db.Set([]byte(key), data)
		} else {
			exist, err := dbMgr.Db.IsExist([]byte(key))
			if !exist {
				if err != nil {
					return err
				}
				return errors.New("key Not Found")
			}
		}

		return dbMgr.Db.Delete([]byte(key))
	} else {
		return errors.New("Unsupported ChainConfig type")
	}
}

func (dbMgr *DbMgr) GetTrxAuthModeByGroupId(groupId string, trxType quorumpb.TrxType, prefix ...string) (quorumpb.TrxAuthMode, error) {
	nodoeprefix := getPrefix(prefix...)
	key := nodoeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + TRX_AUTH_TYPE_PREFIX + "_" + trxType.String()

	//if not specified by group owner
	//follow deny list by default
	//if in deny list, access prohibit
	//if not in deny list, access granted
	isExist, err := dbMgr.Db.IsExist([]byte(key))
	if !isExist {
		return quorumpb.TrxAuthMode_FOLLOW_DNY_LIST, nil
	}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return -1, err
	}

	chainConfigItem := &quorumpb.ChainConfigItem{}
	if err := proto.Unmarshal(value, chainConfigItem); err != nil {
		return -1, err
	}

	trxAuthitem := quorumpb.SetTrxAuthModeItem{}
	perr := proto.Unmarshal(chainConfigItem.Data, &trxAuthitem)
	if perr != nil {
		return -1, perr
	}

	return trxAuthitem.Mode, nil
}

func (dbMgr *DbMgr) GetSendTrxAuthListByGroupId(groupId string, listType quorumpb.AuthListType, prefix ...string) ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error) {
	var chainConfigList []*quorumpb.ChainConfigItem
	var sendTrxRuleList []*quorumpb.ChainSendTrxRuleListItem

	nodeprefix := getPrefix(prefix...)
	var key string
	if listType == quorumpb.AuthListType_ALLOW_LIST {
		key = nodeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + ALLW_LIST_PREFIX
	} else {
		key = nodeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + DENY_LIST_PREFIX
	}
	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		chainConfigItem := quorumpb.ChainConfigItem{}
		err = proto.Unmarshal(v, &chainConfigItem)
		if err != nil {
			return err
		}
		chainConfigList = append(chainConfigList, &chainConfigItem)
		sendTrxRuleListItem := quorumpb.ChainSendTrxRuleListItem{}
		err = proto.Unmarshal(chainConfigItem.Data, &sendTrxRuleListItem)
		if err != nil {
			return err
		}
		sendTrxRuleList = append(sendTrxRuleList, &sendTrxRuleListItem)

		dbmgr_log.Infof("sendTrx %s", sendTrxRuleListItem.Pubkey)

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return chainConfigList, sendTrxRuleList, nil
}

func (dbMgr *DbMgr) CheckTrxTypeAuth(groupId, pubkey string, trxType quorumpb.TrxType, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)

	keyAllow := nodeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + ALLW_LIST_PREFIX + "_" + pubkey
	keyDeny := nodeprefix + CHAIN_CONFIG_PREFIX + "_" + groupId + "_" + DENY_LIST_PREFIX + "_" + pubkey

	isInAllowList, err := dbMgr.Db.IsExist([]byte(keyAllow))
	if err != nil {
		return false, err
	}

	if isInAllowList {
		v, err := dbMgr.Db.Get([]byte(keyAllow))
		chainConfigItem := quorumpb.ChainConfigItem{}
		err = proto.Unmarshal(v, &chainConfigItem)
		if err != nil {
			return false, err
		}

		allowItem := quorumpb.ChainSendTrxRuleListItem{}
		err = proto.Unmarshal(chainConfigItem.Data, &allowItem)
		if err != nil {
			return false, err
		}

		//check if trxType allowed
		for _, allowTrxType := range allowItem.Type {
			if trxType == allowTrxType {
				return true, nil
			}
		}
	}

	isInDenyList, err := dbMgr.Db.IsExist([]byte(keyDeny))
	if err != nil {
		return false, err
	}

	if isInDenyList {
		v, err := dbMgr.Db.Get([]byte(keyDeny))
		chainConfigItem := quorumpb.ChainConfigItem{}
		err = proto.Unmarshal(v, &chainConfigItem)
		if err != nil {
			return false, err
		}

		denyItem := quorumpb.ChainSendTrxRuleListItem{}
		err = proto.Unmarshal(chainConfigItem.Data, &denyItem)
		if err != nil {
			return false, err
		}
		//check if trxType allowed
		for _, denyTrxType := range denyItem.Type {
			if trxType == denyTrxType {
				return false, nil
			}
		}
	}
	trxAuthMode, err := dbMgr.GetTrxAuthModeByGroupId(groupId, trxType, prefix...)
	if err != nil {
		return false, err
	}

	if trxAuthMode == quorumpb.TrxAuthMode_FOLLOW_ALW_LIST {
		//not in allow list, so return false, access denied
		return false, nil
	} else {
		//not in deny list, so return true, access granted
		return true, nil
	}
}

func (dbMgr *DbMgr) UpdateProducerTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return dbMgr.UpdateProducer(trx.Data, prefix)
}

func (dbMgr *DbMgr) UpdateProducer(data []byte, prefix []string) (err error) {
	nodeprefix := getPrefix(prefix...)
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
	return dbMgr.UpdateUser(trx.Data, prefix)
}

func (dbMgr *DbMgr) UpdateUser(data []byte, prefix []string) (err error) {

	nodeprefix := getPrefix(prefix...)

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
	return dbMgr.UpdateAppConfig(trx.Data, Prefix)
}

func (dbMgr *DbMgr) UpdateAppConfig(data []byte, Prefix []string) (err error) {
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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
	nodeprefix := getPrefix(Prefix...)
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

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey
	dbmgr_log.Infof("Add Producer with key %s", key)

	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), pbyte)
}

func (dbMgr *DbMgr) AddProducedBlockCount(groupId, producerPubkey string, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
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

func (dbMgr *DbMgr) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var pList []*quorumpb.ProducerItem
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
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

func (dbMgr *DbMgr) GetUsers(groupId string, prefix ...string) ([]*quorumpb.UserItem, error) {
	var pList []*quorumpb.UserItem
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + USR_PREFIX + "_" + groupId

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
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

func (dbMgr *DbMgr) IsProducer(groupId, producerPubKey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId + "_" + producerPubKey

	//check if group exist
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) UpdateAnnounceTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return dbMgr.UpdateAnnounce(trx.Data, prefix)
}

func (dbMgr *DbMgr) UpdateAnnounce(data []byte, prefix []string) (err error) {
	nodeprefix := getPrefix(prefix...)
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}
	key := nodeprefix + ANN_PREFIX + "_" + item.GroupId + "_" + item.Type.Enum().String() + "_" + item.SignPubkey
	return dbMgr.Db.Set([]byte(key), data)
}

func (dbMgr *DbMgr) GetAnnounceUsersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	nodeprefix := getPrefix(prefix...)
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

func (dbMgr *DbMgr) GetAnnounceProducersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String()
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
	nodeprefix := getPrefix(prefix...)
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
	nodeprefix := getPrefix(prefix...)
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
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + producerSignPubkey
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) IsUserAnnounced(groupId, userSignPubkey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + userSignPubkey
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) UpdateAnnounceResult(announcetype quorumpb.AnnounceType, groupId, signPubkey string, result bool, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + announcetype.String() + "_" + signPubkey

	var pAnnounced *quorumpb.AnnounceItem
	pAnnounced = &quorumpb.AnnounceItem{}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return err
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
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) IsUser(groupId, userPubKey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + userPubKey

	//check if group user (announced) exist
	return dbMgr.Db.IsExist([]byte(key))
}

//update group nonce
func (dbMgr *DbMgr) UpdateNonce(groupId string, prefix ...string) (nonce uint64, err error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + NONCE_PREFIX + "_" + groupId
	seq, err := dbMgr.Db.GetSequence([]byte(key), 100)
	return seq.Next()
}

//get next nonce
func (dbMgr *DbMgr) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + NONCE_PREFIX + "_" + groupId
	seq, err := dbMgr.Db.GetSequence([]byte(key), 100)
	return seq.Next()
}

//update group snapshot
func (dbMgr *DbMgr) UpdateSnapshotTag(groupId string, snapshotTag *quorumpb.SnapShotTag, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SNAPSHOT_PREFIX + "_" + groupId
	value, err := proto.Marshal(snapshotTag)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) GetSnapshotTag(groupId string, prefix ...string) (*quorumpb.SnapShotTag, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SNAPSHOT_PREFIX + "_" + groupId

	//check if item exist
	exist, err := dbMgr.Db.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("SnapshotTag Not Found")
	}

	snapshotTag := quorumpb.SnapShotTag{}
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &snapshotTag)
	return &snapshotTag, err
}

func (dbMgr *DbMgr) UpdateSchema(trx *quorumpb.Trx, prefix ...string) (err error) {
	item := &quorumpb.SchemaItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + item.GroupId + "_" + item.Type

	if item.Action == quorumpb.ActionType_ADD {
		return dbMgr.Db.Set([]byte(key), trx.Data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if item exist
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Announce Not Found")
		}

		return dbMgr.Db.Delete([]byte(key))
	} else {
		err := errors.New("unknow msgType")
		return err
	}
}

func (dbMgr *DbMgr) GetAllSchemasByGroup(groupId string, prefix ...string) ([]*quorumpb.SchemaItem, error) {
	var scmList []*quorumpb.SchemaItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + groupId

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.SchemaItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		scmList = append(scmList, &item)
		return nil
	})

	return scmList, err
}

func (dbMgr *DbMgr) GetSchemaByGroup(groupId, schemaType string, prefix ...string) (*quorumpb.SchemaItem, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + groupId + "_" + schemaType

	schema := quorumpb.SchemaItem{}
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &schema)
	if err != nil {
		return nil, err
	}

	return &schema, err
}

func getPrefix(prefix ...string) string {
	nodeprefix := ""
	if len(prefix) == 1 {
		nodeprefix = prefix[0] + "_"
	}
	return nodeprefix
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

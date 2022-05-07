package chainstorage

import (
	"errors"
	"fmt"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type Storage struct {
	dbmgr *s.DbMgr
}

var storage *Storage
var chaindb_log = logging.Logger("chaindb")

func NewChainStorage(dbmgr *s.DbMgr) (storage *Storage) {
	if storage == nil {
		storage = &Storage{dbmgr}
	}
	return storage
}

//add block
func (cs *Storage) AddBlock(newBlock *quorumpb.Block, cached bool, prefix ...string) error {
	isSaved, err := cs.dbmgr.IsBlockExist(newBlock.BlockId, cached, prefix...)
	if err != nil {
		return err
	}

	if isSaved {
		chaindb_log.Debugf("Block <%s> already saved, ignore", newBlock.BlockId)
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
		pChunk, err := cs.dbmgr.GetBlockChunk(newBlock.PrevBlockId, cached, prefix...)
		if err != nil {
			return err
		}

		//update parent chunk
		pChunk.SubBlockId = append(pChunk.SubBlockId, chunk.BlockId)
		err = cs.dbmgr.SaveBlockChunk(pChunk, cached, prefix...)
		if err != nil {
			return err
		}

		chunk.Height = pChunk.Height + 1     //increase height
		chunk.ParentBlockId = pChunk.BlockId //point to parent
	}

	//save chunk
	return cs.dbmgr.SaveBlockChunk(chunk, cached, prefix...)
}

//remove block
func (cs *Storage) RmBlock(blockId string, cached bool, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + s.BLK_PREFIX + "_" + blockId
	}

	return cs.dbmgr.Db.Delete([]byte(key))
}

func (cs *Storage) UpdateAnnounceResult(announcetype quorumpb.AnnounceType, groupId, signPubkey string, result bool, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + announcetype.String() + "_" + signPubkey

	var pAnnounced *quorumpb.AnnounceItem
	pAnnounced = &quorumpb.AnnounceItem{}

	value, err := cs.dbmgr.Db.Get([]byte(key))
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
	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) UpdateAnnounce(data []byte, prefix ...string) (err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + item.GroupId + "_" + item.Type.Enum().String() + "_" + item.SignPubkey
	return cs.dbmgr.Db.Set([]byte(key), data)
}

//save trx
func (cs *Storage) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.TRX_PREFIX + "_" + trx.TrxId + "_" + fmt.Sprint(trx.Nonce)
	value, err := proto.Marshal(trx)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}

//UNUSED
//rm Trx
func (cs *Storage) RmTrx(trxId string, nonce int64, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.TRX_PREFIX + "_" + trxId + "_" + fmt.Sprint(nonce)
	return cs.dbmgr.Db.Delete([]byte(key))
}

func (cs *Storage) UpdTrx(trx *quorumpb.Trx, prefix ...string) error {
	return cs.AddTrx(trx, prefix...)
}

func (cs *Storage) AddGensisBlock(gensisBlock *quorumpb.Block, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.BLK_PREFIX + "_" + gensisBlock.BlockId

	isExist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}
	if isExist {
		chaindb_log.Debugf("Genesis block <%s> exist, do nothing", gensisBlock.BlockId)
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

	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) GatherBlocksFromCache(newBlock *quorumpb.Block, cached bool, prefix ...string) ([]*quorumpb.Block, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	var blocks []*quorumpb.Block
	blocks = append(blocks, newBlock)
	pointer1 := 0 //point to head
	pointer2 := 0 //point to tail

	pre := nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_"

	for {
		err := cs.dbmgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
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

func (cs *Storage) AddGroup(groupItem *quorumpb.GroupItem) error {
	//check if group exist
	key := s.GROUPITEM_PREFIX + "_" + groupItem.GroupId
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) UpdGroup(groupItem *quorumpb.GroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	key := s.GROUPITEM_PREFIX + "_" + groupItem.GroupId
	//upd group to db
	return cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) RmGroup(item *quorumpb.GroupItem) error {
	//check if group exist
	key := s.GROUPITEM_PREFIX + "_" + item.GroupId
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("Group Not Found")
	}

	//delete group
	return cs.dbmgr.GroupInfoDb.Delete([]byte(key))
}

//update group snapshot
func (cs *Storage) UpdateSnapshotTag(groupId string, snapshotTag *quorumpb.SnapShotTag, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SNAPSHOT_PREFIX + "_" + groupId
	value, err := proto.Marshal(snapshotTag)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) GetSnapshotTag(groupId string, prefix ...string) (*quorumpb.SnapShotTag, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SNAPSHOT_PREFIX + "_" + groupId

	//check if item exist
	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("SnapshotTag Not Found")
	}

	snapshotTag := quorumpb.SnapShotTag{}
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &snapshotTag)
	return &snapshotTag, err
}

func (cs *Storage) UpdateSchema(trx *quorumpb.Trx, prefix ...string) (err error) {
	item := &quorumpb.SchemaItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.SMA_PREFIX + "_" + item.GroupId + "_" + item.Type

	if item.Action == quorumpb.ActionType_ADD {
		return cs.dbmgr.Db.Set([]byte(key), trx.Data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if item exist
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Announce Not Found")
		}

		return cs.dbmgr.Db.Delete([]byte(key))
	} else {
		err := errors.New("unknow msgType")
		return err
	}
}

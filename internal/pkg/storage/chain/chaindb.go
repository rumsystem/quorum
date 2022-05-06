package chainstorage

import (
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

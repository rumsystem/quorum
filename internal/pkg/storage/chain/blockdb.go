package chainstorage

import (
	"errors"
	"strconv"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

//add block
func (cs *Storage) AddBlock(block *quorumpb.Block, cached bool, prefix ...string) error {
	return cs.dbmgr.SaveBlock(block, cached, prefix...)
}

//remove block
func (cs *Storage) RmBlock(groupId string, epoch int64, cached bool, prefix ...string) error {
	return cs.dbmgr.RmBlock(groupId, epoch, cached, prefix...)
}

//get block by block_id
func (cs *Storage) GetBlock(groupId string, epoch int64, cached bool, prefix ...string) (*quorumpb.Block, error) {
	return cs.dbmgr.GetBlock(groupId, epoch, cached, prefix...)
}

func (cs *Storage) GatherBlocksFromCache(block *quorumpb.Block, prefix ...string) ([]*quorumpb.Block, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	var blocks []*quorumpb.Block
	blocks = append(blocks, block)
	epoch := block.Epoch
	pre := nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_" + block.GroupId
	err := cs.dbmgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		b := &quorumpb.Block{}
		perr := proto.Unmarshal(v, b)
		if perr != nil {
			return perr
		}

		epoch++
		if b.GroupId == block.GroupId && b.Epoch == epoch {
			blocks = append(blocks, b)
			return nil
		} else {
			return errors.New("NO_MORE_BLOCK")
		}
	})

	//search done, no more block to attach
	if err.Error() == "NO_MORE_BLOCK" {
		return blocks, nil
	}

	return nil, err
}

func (cs *Storage) AddGensisBlock(gensisBlock *quorumpb.Block, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	epochSD := strconv.FormatInt(gensisBlock.Epoch, 10)
	key := nodeprefix + s.BLK_PREFIX + "_" + epochSD

	isExist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}
	if isExist {
		chaindb_log.Debugf("Genesis block exist, do nothing")
		return nil
	}

	value, err := proto.Marshal(gensisBlock)
	if err != nil {
		return err
	}

	return cs.dbmgr.Db.Set([]byte(key), value)
}

// by cuicat
/*
//try to find the subblocks of one block. search from the block to the to blockid
func (cs *Storage) RepairSubblocksList(blockid, toblockid string, prefix ...string) error {
	if toblockid == blockid {
		return fmt.Errorf("no new blocks, no need to repair")
	}
	blockChunk, err := cs.dbmgr.GetBlockChunk(blockid, false, prefix...)
	if err != nil {
		return err
	}
	blockChunk.SubBlockId = []string{}
	succ := false
	verifyblockid := toblockid
	var dblogger *log.Logger
	logfile, err := os.OpenFile(cs.dbmgr.DataPath+"_blockdbrepair.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logfile.Close()
	if err != nil {
	}
	dblogger = log.New(logfile, "blockdb ", log.LstdFlags)

	dblogger.Printf("verify block: %s", blockid)
	var verifyblockChunk *quorumpb.BlockDbChunk
	for {
		verifyblockChunk, err = cs.dbmgr.GetBlockChunk(verifyblockid, false, prefix...)
		if verifyblockid == blockid {
			break
		}
		if verifyblockChunk == nil {
			dblogger.Printf("the block is nil, id: %s", verifyblockid)
			break
		}
		if verifyblockChunk.ParentBlockId == blockid {
			blockChunk.SubBlockId = append(blockChunk.SubBlockId, verifyblockChunk.BlockId)
			dblogger.Printf("find the subblock of %s the subblockid is %s", blockid, verifyblockChunk.BlockId)
			succ = true
			break
		}
		verifyblockid = verifyblockChunk.ParentBlockId
	}
	if succ == false {
		dblogger.Printf("not find the subblock of %s", blockid)
	} else {
		dblogger.Printf("update the subblockid of %s", blockid)
		err = cs.dbmgr.SaveBlockChunk(blockChunk, false, prefix...)
		if err != nil {
			dblogger.Printf("Error: %s", err)
		}
	}
	return nil
}
*/

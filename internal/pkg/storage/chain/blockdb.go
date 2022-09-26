package chainstorage

import (
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
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

//check if block exist
func (cs *Storage) IsBlockExist(groupId string, epoch int64, cached bool, prefix ...string) (bool, error) {
	return cs.dbmgr.IsBlockExist(groupId, epoch, cached, prefix...)
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
	if err == nil || err.Error() == "NO_MORE_BLOCK" {
		return blocks, nil
	}

	return nil, err
}

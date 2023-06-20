package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type MolassesUser struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
}

var molauser_log = logging.Logger("user")

func (user *MolassesUser) NewUser(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molauser_log.Debugf("NewUser called")
	user.grpItem = item
	user.nodename = nodename
	user.cIface = iface
	user.groupId = item.GroupId
}

func (user *MolassesUser) AddBlock(block *quorumpb.Block) error {
	molauser_log.Debugf("<%s> AddBlock called, BlockId <%d>", user.groupId, block.BlockId)

	//check if block already exist in chain
	isBlockExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.BlockId, false, user.nodename)
	if err != nil {
		return err
	}
	if isBlockExist {
		//block already on chain, ignore
		molauser_log.Debugf("<%s> block <%d> already on chain, ignore", user.groupId, block.BlockId)
		return nil
	}

	//try add new block
	parentBlockId := block.BlockId - 1
	isParentOnChain, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, parentBlockId, false, user.nodename)
	if !isParentOnChain {
		molauser_log.Debugf("<%s> parent block <%d> not valid, save this block to cache, Trxs inside this block ARE NOT APPLIED", user.groupId, parentBlockId)
		err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
		if err != nil {
			return err
		}

		return nil
	}
	//get parent block
	parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, parentBlockId, false, user.nodename)
	if err != nil {
		molauser_log.Errorf("<%s> Get Parent Block failed, err <%s>", user.groupId, err.Error())
		return err
	}

	//valid block with parent block
	valid, err := rumchaindata.ValidBlockWithParent(block, parentBlock)
	if err != nil {
		molauser_log.Errorf("<%s> ValidBlockWithParent failed, err <%s>", user.groupId, err.Error())
		return err
	}

	if !valid {
		molauser_log.Warningf("<%s> invalid block <%s>, ignore", user.groupId, err.Error())
		return fmt.Errorf("invalid block")
	}

	molauser_log.Debugf("block is validated, save it to chain")
	err = user.saveBlock(block, false)
	if err != nil {
		molauser_log.Errorf("<%s> save block failed, err <%s>", user.groupId, err.Error())
		return err
	}

	//search if any cached block can be chainned with this block
	currBlock := block
	for {
		nextBlockId := currBlock.BlockId + 1
		nextBlockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(currBlock.GroupId, nextBlockId, true, user.nodename)
		if !nextBlockExist {
			//next block not exist, break
			break
		}

		//get next block
		nextBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, nextBlockId, true, user.nodename)
		if err != nil {
			molaproducer_log.Warningf("get next block failed with error: %s", err.Error())
			break
		}

		//valid next block with currblock
		valid, err := rumchaindata.ValidBlockWithParent(nextBlock, currBlock)
		if err != nil {
			molaproducer_log.Warningf("valid next block failed with error: %s", err.Error())
			break
		}

		if !valid {
			molaproducer_log.Warningf("<%s> invalid block <%s>, ignore", user.groupId, err.Error())
			break
		}

		//move block from cache to chain and apply all trxs
		err = user.saveBlock(nextBlock, true)
		if err != nil {
			molaproducer_log.Warningf("save next block failed with error: %s", err.Error())
			break
		}

		//start next round
		currBlock = nextBlock
	}

	molauser_log.Debugf("<%s> AddBlock done", user.groupId)
	return nil
}

func (user *MolassesUser) saveBlock(block *quorumpb.Block, rmFromCache bool) error {
	//add block to chain
	if rmFromCache {
		molauser_log.Debugf("<%s> move block <%d> from cache to chain", user.groupId, block.BlockId)
		err := nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.BlockId, true, user.nodename)
		if err != nil {
			return err
		}
	}

	molauser_log.Debugf("<%s> add block <%d> to chain", user.groupId, block.BlockId)
	err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, false, user.nodename)
	if err != nil {
		return err
	}

	//apply trxs
	molauser_log.Debugf("<%s> apply trxs", user.groupId)
	err = user.cIface.ApplyTrxsFullNode(block.Trxs, user.nodename)
	if err != nil {
		molauser_log.Errorf("apply trxs failed with error: %s", err.Error())
		return err
	}

	//update chain info
	molauser_log.Debugf("<%s> UpdChainInfo, upd highest blockId from <%d> to <%d>", user.groupId, user.cIface.GetCurrBlockId(), block.BlockId)
	user.cIface.SetCurrBlockId(block.BlockId)
	//user.cIface.SetCurrEpoch(block.Epoch)
	user.cIface.SetLastUpdate(block.TimeStamp)
	user.cIface.SaveChainInfoToDb()

	return nil
}

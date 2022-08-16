package consensus

import (
	"errors"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type MolassesUser struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
}

var molauser_log = logging.Logger("user")

func (user *MolassesUser) Init(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molauser_log.Debugf("Init called")
	user.grpItem = item
	user.nodename = nodename
	user.cIface = iface
	user.groupId = item.GroupId
	molauser_log.Infof("<%s> User created", user.groupId)
}

func (user *MolassesUser) AddBlock(block *quorumpb.Block) error {
	molauser_log.Debugf("<%s> AddBlock called", user.groupId)

	//check if parent of block exist
	parentExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch-1, false, user.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		molauser_log.Debugf("<%s> parent of block <%d> is not exist", user.groupId, block.Epoch-1)

		//check if block is in cache
		isCached, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, true, user.nodename)
		if err != nil {
			return err
		}

		if !isCached {
			molauser_log.Debugf("<%s> add block to catch", user.groupId)
			//Save block to cache
			err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
			if err != nil {
				return err
			}
		}

		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, block.Epoch-1, false, user.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := rumchaindata.IsBlockValid(block, parentBlock)
	if !valid {
		molauser_log.Warningf("<%s> invalid block <%s>", user.groupId, err.Error())
		molauser_log.Debugf("<%s> remove invalid block <%d> from cache", user.groupId, block.Epoch)
		return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.Epoch, true, user.nodename)
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, user.nodename)
	if err != nil {
		return err
	}

	//get all trxs from blocks
	var trxs []*quorumpb.Trx
	trxs, err = rumchaindata.GetAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply trxs
	err = user.cIface.ApplyUserTrxs(trxs, user.nodename)
	if err != nil {
		return err
	}

	//move gathered blocks from cache to chain
	for _, block := range blocks {
		molauser_log.Debugf("<%s> move block <%d> from cache to chain", user.groupId, block.Epoch)
		err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, false, user.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.Epoch, true, user.nodename)
		if err != nil {
			return err
		}
	}

	return user.cIface.UpdChainInfo(block.Epoch)
}

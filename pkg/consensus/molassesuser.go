package consensus

import (
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
	molauser_log.Debugf("<%s> AddBlock called, epoch <%d>", user.groupId, block.Epoch)
	var blocks []*quorumpb.Block

	//check if block exist
	blockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, false, user.nodename)
	if blockExist { // check if we need to apply trxs again
		// block already saved
		molauser_log.Debugf("Block exist")
		blocks = append(blocks, block)
	} else {
		//check if block cached
		isBlockCatched, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, true, user.nodename)

		//check if block parent exist
		parentEpoch := block.Epoch - 1
		parentExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, parentEpoch, false, user.nodename)

		if !parentExist {
			if isBlockCatched {
				molauser_log.Debugf("Block already catched but parent not exist, wait more block")
				return nil
			} else {
				molauser_log.Debugf("parent of block <%d> is not exist and block not catched, catch it.", block.Epoch)
				//add this block to cache
				err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
				if err != nil {
					return err
				}
			}
		} else {
			//get parent block
			parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, parentEpoch, false, user.nodename)
			if err != nil {
				return err
			}

			//valid block with parent block
			valid, err := rumchaindata.IsBlockValid(block, parentBlock)
			if !valid {
				molauser_log.Warningf("<%s> invalid block <%s>", user.groupId, err.Error())
				molauser_log.Debugf("<%s> remove invalid block <%d> from cache", user.groupId, block.Epoch)
				return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.Epoch, true, user.nodename)
			} else {
				molauser_log.Debugf("block is validated")
			}

			//add this block to cache
			err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
			if err != nil {
				return err
			}

			//search cache, gather all blocks can be connected with this block
			blockfromcache, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, user.nodename)
			if err != nil {
				return err
			}

			blocks = append(blocks, blockfromcache...)

			//move collected blocks from cache to chain
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
		}
	}

	if block.Epoch > user.cIface.GetCurrEpoch() {
		//update latest group epoch
		molauser_log.Debugf("<%s> UpdChainInfo, upd highest epoch from <%d> to <%d>", user.groupId, user.cIface.GetCurrEpoch(), block.Epoch)
		user.cIface.SetCurrEpoch(block.Epoch)
		user.cIface.SetLastUpdate(block.TimeStamp)
		user.cIface.SaveChainInfoToDb()
	} else {
		molauser_log.Debugf("<%s> No need to update highest Epoch", user.groupId)
	}

	//get all trxs from blocks
	var trxs []*quorumpb.Trx
	trxs, err := rumchaindata.GetAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply trxs
	return user.cIface.ApplyTrxsFullNode(trxs, user.nodename)
}

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

	//check if block exist
	blockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, false, user.nodename)
	if blockExist { // check if we need to apply trxs again
		// block already saved
		molauser_log.Debugf("Block exist, ignore")
	} else {
		//check if block cached
		isBlockCatched, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, true, user.nodename)

		//check if block parent exist
		parentEpoch := block.Epoch - 1
		parentExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, parentEpoch, false, user.nodename)

		if !parentExist {
			if isBlockCatched {
				molauser_log.Debugf("Block already catched but parent not exist, wait more block")
				return nil
			} else {
				molauser_log.Debugf("parent of block <%d> is not exist and block not catched, catch it.", block.Epoch)
				//add this block to cache
				err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
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
			valid, err := rumchaindata.ValidBlockWithParent(block, parentBlock)
			if !valid {
				molauser_log.Warningf("<%s> invalid block <%s>", user.groupId, err.Error())
				molauser_log.Debugf("<%s> remove invalid block <%d> from cache", user.groupId, block.Epoch)
				return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.Epoch, true, user.nodename)
			} else {
				molauser_log.Debugf("block is validated")
			}

			//add this block to cache
			if !isBlockCatched {
				err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
				if err != nil {
					return err
				}
			}

			//search cache, gather all blocks can be connected with this block (this block is the first one in the returned block list)
			blockfromcache, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, user.nodename)
			if err != nil {
				return err
			}

			//move collected blocks from cache to chain
			for _, bc := range blockfromcache {
				molauser_log.Debugf("<%s> move block <%d> from cache to chain", user.groupId, bc.Epoch)
				err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(bc, false, user.nodename)
				if err != nil {
					return err
				}

				err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(bc.GroupId, bc.Epoch, true, user.nodename)
				if err != nil {
					return err
				}

				if bc.Epoch > user.cIface.GetCurrEpoch() {
					//update latest group epoch
					molauser_log.Debugf("<%s> UpdChainInfo, upd highest epoch from <%d> to <%d>", user.groupId, user.cIface.GetCurrEpoch(), bc.Epoch)
					user.cIface.SetCurrEpoch(bc.Epoch)
					user.cIface.SetLastUpdate(bc.TimeStamp)
					user.cIface.SaveChainInfoToDb()
				} else {
					molauser_log.Debugf("<%s> No need to update highest Epoch", user.groupId)
				}
			}

			//get all trxs from blocks
			var trxs []*quorumpb.Trx
			trxs, err = rumchaindata.GetAllTrxs(blockfromcache)
			if err != nil {
				return err
			}

			//apply trxs
			return user.cIface.ApplyTrxsFullNode(trxs, user.nodename)
		}
	}
	return nil

}

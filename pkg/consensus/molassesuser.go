package consensus

import (
	"errors"

	"github.com/rumsystem/quorum/internal/pkg/conn"
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

	//check if block is in storage
	isSaved, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.BlockId, false, user.nodename)
	if err != nil {
		return err
	}

	if isSaved {
		return errors.New("Block already saved, ignore")
	}

	//check if block is in cache
	isCached, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.BlockId, true, user.nodename)
	if err != nil {
		return err
	}

	if isCached {
		molauser_log.Debugf("<%s> cached block, update block", user.groupId)
	}

	//Save block to cache
	err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, user.nodename)
	if err != nil {
		return err
	}

	//check if parent of block exist
	parentExist, err := nodectx.GetNodeCtx().GetChainStorage().IsParentExist(block.PrevBlockId, false, user.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		molauser_log.Debugf("<%s> parent of block <%s> is not exist", user.groupId, block.BlockId)
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.PrevBlockId, false, user.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := rumchaindata.IsBlockValid(block, parentBlock)
	if !valid {
		molauser_log.Debugf("<%s> remove invalid block <%s> from cache", user.groupId, block.BlockId)
		molauser_log.Warningf("<%s> invalid block <%s>", user.groupId, err.Error())
		return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.BlockId, true, user.nodename)
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, true, user.nodename)
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
		molauser_log.Debugf("<%s> move block <%s> from cache to chain", user.groupId, block.BlockId)
		err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, false, user.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.BlockId, true, user.nodename)
		if err != nil {
			return err
		}
	}
	//update block produced count
	for _, block := range blocks {
		err := nodectx.GetNodeCtx().GetChainStorage().AddProducedBlockCount(user.groupId, block.ProducerPubKey, user.nodename)
		if err != nil {
			return err
		}
	}

	//calculate new height
	molauser_log.Debugf("<%s> height before recal <%d>", user.groupId, user.grpItem.HighestHeight)
	topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(user.grpItem.HighestBlockId, false, user.nodename)
	if err != nil {
		return err
	}
	newHeight, newHighestBlockId, err := user.cIface.RecalChainHeight(blocks, user.grpItem.HighestHeight, topBlock, user.nodename)
	if err != nil {
		return err
	}
	molauser_log.Debugf("<%s> new height <%d>, new highest blockId %v", user.groupId, newHeight, newHighestBlockId)

	//if the new block is not highest block after recalculate, we need to "trim" the chain
	if newHeight < user.grpItem.HighestHeight {

		//from parent of the new blocks, get all blocks not belong to the longest path
		resendBlocks, err := user.cIface.GetTrimedBlocks(blocks, user.nodename)
		if err != nil {
			return err
		}

		var resendTrxs []*quorumpb.Trx
		resendTrxs, err = user.cIface.GetMyTrxs(resendBlocks, user.nodename, user.grpItem.UserSignPubkey)

		if err != nil {
			return err
		}

		UpdateResendCount(resendTrxs)
		err = user.resendTrx(resendTrxs)
	}

	return user.cIface.UpdChainInfo(newHeight, newHighestBlockId)
}

func (user *MolassesUser) sendTrx(trx *quorumpb.Trx, channel conn.PsConnChanel) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(user.groupId)
	if err != nil {
		return "", err
	}

	err = connMgr.SendTrxPubsub(trx, channel)
	if err != nil {
		return "", err
	}

	err = user.cIface.GetPubqueueIface().TrxEnqueue(user.groupId, trx)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

//resend all trx in the list
func (user *MolassesUser) resendTrx(trxs []*quorumpb.Trx) error {
	molauser_log.Debugf("<%s> resendTrx called", user.groupId)
	for _, trx := range trxs {
		molauser_log.Debugf("<%s> resend Trx <%s>", user.groupId, trx.TrxId)
		user.sendTrx(trx, conn.ProducerChannel)
	}
	return nil
}

//update resend count (+1) for all trxs
func UpdateResendCount(trxs []*quorumpb.Trx) ([]*quorumpb.Trx, error) {
	for _, trx := range trxs {
		trx.ResendCount++
	}
	return trxs, nil
}

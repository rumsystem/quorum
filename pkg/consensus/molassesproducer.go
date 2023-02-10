package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var molaproducer_log = logging.Logger("producer")

type MolassesProducer struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
	bft      *TrxBft
}

func (producer *MolassesProducer) NewProducer(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molaproducer_log.Debug("NewProducer called")
	producer.grpItem = item
	producer.cIface = iface
	producer.nodename = nodename
	producer.groupId = item.GroupId

	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Error("create bft failed")
		molaproducer_log.Error(err.Error())
		return
	}
	producer.bft = NewTrxBft(*config, producer)
}

func (producer *MolassesProducer) TryPropose() {
	molaproducer_log.Debug("TryPropose called")
	newEpoch := producer.cIface.GetCurrEpoch() + 1
	producer.bft.propose(newEpoch)
}

func (producer *MolassesProducer) RecreateBft() {
	molaproducer_log.Debug("RecreateBft called")
	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Errorf("recreate bft failed")
		molaproducer_log.Error(err.Error())
		return
	}

	producer.bft = NewTrxBft(*config, producer)
}

func (producer *MolassesProducer) createBftConfig() (*Config, error) {
	molaproducer_log.Debugf("<%s> createBftConfig called", producer.groupId)
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(producer.groupId, producer.nodename)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, producer := range producer_nodes {
		nodes = append(nodes, producer.ProducerPubkey)
	}

	molaproducer_log.Debugf("Get <%d> producers", len(nodes))
	for _, producerId := range nodes {
		molaproducer_log.Debugf(">>> producer_id <%s>", producerId)
	}

	N := len(nodes)
	f := (N - 1) / 3 //f * 3 < N

	molaproducer_log.Debugf("Failable node <%d>", f)

	//use fixed scalar size
	scalar := 20
	//batchSize := (len(nodes) * 2) * scalar
	batchSize := scalar

	molaproducer_log.Debugf("batchSize <%d>", batchSize)

	config := &Config{
		N:         N,
		f:         f,
		Nodes:     nodes,
		BatchSize: batchSize,
		MyPubkey:  producer.grpItem.UserSignPubkey,
	}

	return config, nil
}

// Add Block will be called when producer sync with other producer node
func (producer *MolassesProducer) AddBlock(block *quorumpb.Block) error {
	molaproducer_log.Debugf("<%s> AddBlock called, epoch <%d>", producer.groupId, block.Epoch)

	//check if block exist
	blockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, false, producer.nodename)
	if blockExist { // check if we need to apply trxs again
		// block already saved
		molaproducer_log.Debugf("Block exist, ignore")

	} else {
		//check if block cached
		isBlockCatched, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, true, producer.nodename)

		//check if block parent exist
		parentEpoch := block.Epoch - 1
		parentExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, parentEpoch, false, producer.nodename)

		if !parentExist {
			if isBlockCatched {
				molaproducer_log.Debugf("Block already catched but parent not exist, wait more block")
				return nil
			} else {
				molaproducer_log.Debugf("parent of block <%d> is not exist and block not catched, catch it.", block.Epoch)
				//add this block to cache
				err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, producer.nodename)
				if err != nil {
					return err
				}
			}
		} else {
			//get parent block
			parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, parentEpoch, false, producer.nodename)
			if err != nil {
				return err
			}

			//valid block with parent block
			valid, err := rumchaindata.IsBlockValid(block, parentBlock)
			if !valid {
				molaproducer_log.Warningf("<%s> invalid block <%s>", producer.groupId, err.Error())
				molaproducer_log.Debugf("<%s> remove invalid block <%d> from cache", producer.groupId, block.Epoch)
				return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.Epoch, true, producer.nodename)
			} else {
				molaproducer_log.Debugf("block is validated")
			}

			//add this block to cache
			if !isBlockCatched {
				err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, producer.nodename)
				if err != nil {
					return err
				}
			}

			//search cache, gather all blocks can be connected with this block (this block is the first one in the returned block list)
			blockfromcache, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, producer.nodename)
			if err != nil {
				return err
			}

			//move collected blocks from cache to chain
			for _, blk := range blockfromcache {
				molaproducer_log.Debugf("<%s> move block <%d> from cache to chain", producer.groupId, blk.Epoch)
				err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(blk, false, producer.nodename)
				if err != nil {
					return err
				}

				err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(blk.GroupId, blk.Epoch, true, producer.nodename)
				if err != nil {
					return err
				}

				if blk.Epoch > producer.cIface.GetCurrEpoch() {
					//update latest group epoch
					molaproducer_log.Debugf("<%s> UpdChainInfo, upd highest epoch from <%d> to <%d>", producer.groupId, producer.cIface.GetCurrEpoch(), blk.Epoch)
					producer.cIface.SetCurrEpoch(blk.Epoch)
					producer.cIface.SetLastUpdate(blk.TimeStamp)
					producer.cIface.SaveChainInfoToDb()
				} else {
					molaproducer_log.Debugf("<%s> No need to update highest Epoch", producer.groupId)
				}
			}

			//get all trxs from blocks
			var trxs []*quorumpb.Trx
			trxs, err = rumchaindata.GetAllTrxs(blockfromcache)
			if err != nil {
				return err
			}

			//apply trxs
			return producer.cIface.ApplyTrxsProducerNode(trxs, producer.nodename)
		}
	}
	return nil
}

func (producer *MolassesProducer) AddTrx(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> AddTrx called", producer.groupId)

	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, producer.nodename)
	if err != nil {
		return
	}

	if !isAllow {
		molaproducer_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", producer.groupId, trx.SenderPubkey, trx.Type.String())
		return
	}

	//check if trx with same nonce exist, !!Only applied to client which support nonce
	isExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, trx.Nonce, producer.nodename)
	if isExist {
		molaproducer_log.Debugf("<%s> Trx <%s> with nonce <%d> already packaged, ignore", producer.groupId, trx.TrxId, trx.Nonce)
		return
	}

	/*
		if trx.SudoTrx {
			molaproducer_log.Debugf("<%s> Molasses AddTrx called, add sudo trx <%s>", producer.groupId, trx.TrxId)
			err = producer.bft.AddTrx(trx)
			if err != nil {
				molaproducer_log.Errorf("add trx failed %s", err.Error())
			}
		} else {

	*/
	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err = producer.bft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed %s", err.Error())
	}
}

func (producer *MolassesProducer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//molaproducer_log.Debugf("<%s> HandleHBMsg, Epoch <%d>", producer.groupId, hbmsg.Epoch)
	return producer.bft.HandleMessage(hbmsg)
}

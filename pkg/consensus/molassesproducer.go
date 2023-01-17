package consensus

import (
	"errors"

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
		molaproducer_log.Debugf(">>> producer_id %s", producerId)
	}

	N := len(nodes)
	f := (N - 1) / 3 //f * 3 < N

	molaproducer_log.Debugf("Failable node %d", f)

	//use fixed scalar size
	scalar := 20
	//batchSize := (len(nodes) * 2) * scalar
	batchSize := scalar

	molaproducer_log.Debugf("batchSize %d", batchSize)

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
	molaproducer_log.Debugf("<%s> AddBlock called", producer.groupId)
	var blocks []*quorumpb.Block

	//check if block exist
	blockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, false, producer.nodename)
	if blockExist { // check if we need to apply trxs again
		// block already saved
		// maybe saved by local producer or during sync, receive this block from someone else
		molaproducer_log.Debugf("Block exist")
		blocks = append(blocks, block)
	} else { //block not exist, we don't have local producer
		//check if parent of block exist
		molaproducer_log.Debugf("Block not exist")
		parentExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch-1, false, producer.nodename)
		if err != nil {
			return err
		}

		if !parentExist {
			molaproducer_log.Debugf("<%s> parent of block <%d> is not exist", producer.groupId, block.Epoch-1)

			//check if block is in cache
			isCached, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.Epoch, true, producer.nodename)
			if err != nil {
				return err
			}

			if !isCached {
				molaproducer_log.Debugf("<%s> add block to catch", producer.groupId)
				//Save block to cache
				err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, producer.nodename)
				if err != nil {
					return err
				}
			}

			return errors.New("PARENT_NOT_EXIST")
		}

		//get parent block
		parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, block.Epoch-1, false, producer.nodename)
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
		err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, producer.nodename)
		if err != nil {
			return err
		}

		//search cache, gather all blocks can be connected with this block
		blockfromcache, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, producer.nodename)
		if err != nil {
			return err
		}

		blocks = append(blocks, blockfromcache...)

		//move collected blocks from cache to chain
		for _, block := range blocks {
			molaproducer_log.Debugf("<%s> move block <%d> from cache to chain", producer.groupId, block.Epoch)
			err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, false, producer.nodename)
			if err != nil {
				return err
			}

			err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.Epoch, true, producer.nodename)
			if err != nil {
				return err
			}
		}
	}

	//update latest epoch only if epoch of block is larger than current group epoch
	if block.Epoch > producer.cIface.GetCurrEpoch() {
		producer.cIface.SetCurrEpoch(block.Epoch)
		producer.cIface.SetLastUpdate(block.TimeStamp)
		producer.cIface.SaveChainInfoToDb()
	}

	//get all trxs from blocks
	var trxs []*quorumpb.Trx
	trxs, err := rumchaindata.GetAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply trxs
	return producer.cIface.ApplyTrxsProducerNode(trxs, producer.nodename)
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
	//molaproducer_log.Debugf("<%s> HandleHBMsg <%s>", producer.groupId, hbmsg.Epoch)
	return producer.bft.HandleMessage(hbmsg)
}

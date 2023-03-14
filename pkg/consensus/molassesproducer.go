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
	ptbft    *PTBft
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
	producer.ptbft = NewPTBft(*config, producer)
}

func (producer *MolassesProducer) StartPropose() {
	molaproducer_log.Debug("StartPropose called")
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(producer.groupId, producer.nodename)
	if err != nil {
		return
	}

	isProducer := false
	for _, p := range producer_nodes {
		if producer.grpItem.UserSignPubkey == p.ProducerPubkey {
			isProducer = true
			break
		}
	}

	if isProducer {
		molaproducer_log.Debug("approved producer start propose")
		producer.ptbft.Start()
	} else {
		molaproducer_log.Debug("unapproved producer do nothing")
	}
}

func (producer *MolassesProducer) StopPropose() {
	molaproducer_log.Debug("StopPropose called")
	if producer.ptbft != nil {
		producer.ptbft.Stop()
	}
}

func (producer *MolassesProducer) RecreateBft() {
	molaproducer_log.Debug("RecreateBft called")

	//stop current bft
	if producer.ptbft != nil {
		producer.ptbft.Stop()
	}

	//check if I am still a valid producer
	if !producer.cIface.IsProducer() {
		molaproducer_log.Debug("no longer approved producer, quit bft")
		return
	}

	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Errorf("recreate bft failed")
		molaproducer_log.Error(err.Error())
		return
	}

	producer.ptbft = NewPTBft(*config, producer)
	producer.ptbft.Start()
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
	molaproducer_log.Debugf("<%s> AddBlock called, BlockId <%d>", producer.groupId, block.BlockId)

	//check if block exist
	blockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.BlockId, false, producer.nodename)
	if blockExist { // check if we need to apply trxs again
		// block already saved
		molaproducer_log.Debugf("Block exist, ignore")
	} else {
		//check if block cached
		isBlockCatched, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.BlockId, true, producer.nodename)

		//check if block parent exist
		parentBlockId := block.BlockId - 1
		parentExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, parentBlockId, false, producer.nodename)

		if !parentExist {
			if isBlockCatched {
				molaproducer_log.Debugf("Block already catched but parent is not exist, wait more block to fill the gap")
				return nil
			} else {
				molaproducer_log.Debugf("parent of block <%d> is not exist and block not catched, catch it.", block.BlockId)
				//add this block to cache
				err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, producer.nodename)
				if err != nil {
					return err
				}
			}
		} else {
			//get parent block
			parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, parentBlockId, false, producer.nodename)
			if err != nil {
				return err
			}

			//valid block with parent block
			valid, err := rumchaindata.ValidBlockWithParent(block, parentBlock)
			if !valid {
				molaproducer_log.Warningf("<%s> invalid block <%s>", producer.groupId, err.Error())
				molaproducer_log.Debugf("<%s> remove invalid block <%d> from cache", producer.groupId, block.Epoch)
				return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.BlockId, true, producer.nodename)
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
				molaproducer_log.Debugf("<%s> move block <%d> from cache to chain", producer.groupId, blk.BlockId)
				err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(blk, false, producer.nodename)
				if err != nil {
					return err
				}

				err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(blk.GroupId, blk.BlockId, true, producer.nodename)
				if err != nil {
					return err
				}

				if blk.BlockId > producer.cIface.GetCurrBlockId() {
					//update latest group info
					molaproducer_log.Debugf("<%s> UpdChainInfo, blockId from <%d> to <%d>",
						producer.groupId, producer.cIface.GetCurrBlockId(), blk.BlockId)
					producer.cIface.SetCurrBlockId(blk.BlockId)
					producer.cIface.SetLastUpdate(blk.TimeStamp)
				}

				if blk.Epoch > producer.cIface.GetCurrEpoch() {
					molaproducer_log.Debugf("<%s> UpdChainInfo, Epoch from <%d> to <%d>",
						producer.groupId, producer.cIface.GetCurrEpoch(), blk.Epoch)
					producer.cIface.SetCurrEpoch(blk.Epoch)
					producer.cIface.SaveChainInfoToDb()
					/*
						if producer.bft.CurrTask != nil && producer.bft.CurrTask.Epoch <= blk.Epoch {
							producer.bft.KillAndRunNextRound()
						}
					*/
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

	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err = producer.ptbft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed %s", err.Error())
	}
}

func (producer *MolassesProducer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//molaproducer_log.Debugf("<%s> HandleHBMsg, Epoch <%d>", producer.groupId, hbmsg.Epoch)
	return producer.ptbft.HandleMessage(hbmsg)
}

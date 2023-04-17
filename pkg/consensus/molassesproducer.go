package consensus

import (
	"context"
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var molaproducer_log = logging.Logger("producer")

type MolassesProducer struct {
	groupId  string
	nodename string
	grpItem  *quorumpb.GroupItem
	cIface   def.ChainMolassesIface

	ptbft  *PTBft
	ctx    context.Context
	locker sync.RWMutex
}

func (producer *MolassesProducer) NewProducer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molaproducer_log.Debug("<%s> NewProducer called", item.GroupId)
	producer.nodename = nodename
	producer.groupId = item.GroupId
	producer.grpItem = item
	producer.cIface = iface
	producer.ctx = ctx
}

func (producer *MolassesProducer) StartPropose() {
	molaproducer_log.Debug("StartPropose called")

	producer.locker.Lock()
	defer producer.locker.Unlock()

	if !producer.cIface.IsProducer() {
		molaproducer_log.Debug("unapproved producer do nothing")
	}

	molaproducer_log.Debug("producer <%s >start propose")
	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Error("create bft failed")
		molaproducer_log.Error(err.Error())
		return
	}

	producer.ptbft = NewPTBft(producer.ctx, *config, producer.cIface)
	producer.ptbft.Start()
}

func (producer *MolassesProducer) StopPropose() {
	molaproducer_log.Debug("StopPropose called")
	producer.locker.Lock()
	defer producer.locker.Unlock()

	if producer.ptbft != nil {
		producer.ptbft.Stop()
	}
	producer.ptbft = nil
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
		GroupId:     producer.groupId,
		NodeName:    producer.nodename,
		MyPubkey:    producer.grpItem.UserSignPubkey,
		OwnerPubKey: producer.grpItem.OwnerPubKey,

		N:         N,
		f:         f,
		Nodes:     nodes,
		BatchSize: batchSize,
	}

	return config, nil
}

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
		molaproducer_log.Debugf("<%s> pubkey <%s> don't has permission to send trx with type <%s>", producer.groupId, trx.SenderPubkey, trx.Type.String())
		return
	}

	//check if trx with same trxid exist (already packaged)
	isExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, producer.nodename)
	if isExist {
		molaproducer_log.Debugf("<%s> trx <%s> with nonce <%d> already packaged, ignore", producer.groupId, trx.TrxId, trx.Nonce)
		return
	}

	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err = producer.ptbft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed with error <%s>", err.Error())
	}
}

func (producer *MolassesProducer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//molaproducer_log.Debugf("<%s> HandleHBMsg, Epoch <%d>", producer.groupId, hbmsg.Epoch)
	if producer.ptbft != nil {
		producer.ptbft.HandleMessage(hbmsg)
	}
	return nil
}

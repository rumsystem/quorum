package consensus

import (
	"context"
	"fmt"
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
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
	molaproducer_log.Debugf("<%s> NewProducer called", item.GroupId)
	producer.nodename = nodename
	producer.groupId = item.GroupId
	producer.grpItem = item
	producer.cIface = iface
	producer.ctx = ctx
}

func (producer *MolassesProducer) StartPropose() {
	molaproducer_log.Debugf("<%s> StartPropose called", producer.groupId)

	producer.locker.Lock()
	defer producer.locker.Unlock()

	if !producer.cIface.IsProducer() {
		molaproducer_log.Debugf("<%s> unapproved producer do nothing", producer.groupId)
		return
	}

	molaproducer_log.Debugf("<%s> producer <%s> start propose", producer.groupId, producer.grpItem.UserSignPubkey)
	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Error("create bft failed with error: %s", err.Error())
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
	blockExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, block.BlockId, false, producer.nodename)
	if err != nil {
		return err
	}

	if blockExist {
		// block already on chain, ignore
		molaproducer_log.Debugf("<%s> block <%d> already on chain, ignore", producer.groupId, block.BlockId)
		return nil
	}

	//try add block to chain
	//check if block parent exist
	parentBlockId := block.BlockId - 1
	isParentOnChain, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.GroupId, parentBlockId, false, producer.nodename)
	if !isParentOnChain {
		molaproducer_log.Debugf("parent of block <%d> not valid, save this block to cache, Trxs inside this block ARE NOT APPLIED", block.BlockId)
		err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, producer.nodename)
		if err != nil {
			return err
		}
		return nil
	}

	//get parent block
	parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, parentBlockId, false, producer.nodename)
	if err != nil {
		molaproducer_log.Warningf("<%s> get parent block failed with error: %s", producer.groupId, err.Error())
		return err
	}

	//valid block with parent block
	valid, err := rumchaindata.ValidBlockWithParent(block, parentBlock)

	if err != nil {
		molaproducer_log.Warningf("<%s> valid block failed with error: %s", producer.groupId, err.Error())
		return err
	}

	if !valid {
		molaproducer_log.Warningf("<%s> invalid block <%s>, ignore", producer.groupId, err.Error())
		return fmt.Errorf("invalid block")
	}

	molaproducer_log.Debugf("block is validated, save it to chain")
	err = producer.saveBlock(block, false)
	if err != nil {
		return err
	}

	//search if any cached block can be chainned with this block
	currBlock := block
	for {
		nextBlockId := currBlock.BlockId + 1
		nextBlockExist, _ := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(currBlock.GroupId, nextBlockId, true, producer.nodename)
		if !nextBlockExist {
			//next block not exist, break
			break
		}

		//get next block
		nextBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.GroupId, nextBlockId, true, producer.nodename)
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
			molaproducer_log.Warningf("<%s> invalid block <%s>, ignore", producer.groupId, err.Error())
			break
		}

		//move block from cache to chain and apply all trxs
		err = producer.saveBlock(nextBlock, true)
		if err != nil {
			molaproducer_log.Warningf("save next block failed with error: %s", err.Error())
			break
		}

		//start next round
		currBlock = nextBlock
	}

	molaproducer_log.Debugf("<%s> AddBlock done", producer.groupId)
	return nil
}

func (producer *MolassesProducer) saveBlock(block *quorumpb.Block, rmFromCache bool) error {
	//add block to chain
	if rmFromCache {
		molaproducer_log.Debugf("<%s> move block <%d> from cache to chain", producer.groupId, block.BlockId)
		err := nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.GroupId, block.BlockId, true, producer.nodename)
		if err != nil {
			return err
		}
	}

	molaproducer_log.Debugf("<%s> add block <%d> to chain", producer.groupId, block.BlockId)
	err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, false, producer.nodename)
	if err != nil {
		return err
	}

	//apply trxs
	molauser_log.Debugf("<%s> apply trxs", producer.groupId)
	//err = producer.cIface.ApplyTrxsProducerNode(block.Trxs, producer.nodename)
	if err != nil {
		molaproducer_log.Errorf("apply trxs failed with error: %s", err.Error())
		return err
	}

	//update chain info
	molauser_log.Debugf("<%s> UpdChainInfo, upd highest blockId from <%d> to <%d>", producer.groupId, producer.cIface.GetCurrBlockId(), block.BlockId)
	producer.cIface.SetCurrBlockId(block.BlockId)
	producer.cIface.SetLastUpdate(block.TimeStamp)
	producer.cIface.SaveChainInfoToDb()

	return nil
}

func (producer *MolassesProducer) AddTrxToTxBuffer(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> AddTrxToTxBuffer called", producer.groupId)

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
		molaproducer_log.Debugf("<%s> trx <%s> already packaged, ignore", producer.groupId, trx.TrxId)
		return
	}

	//molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err = producer.ptbft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed with error <%s>", err.Error())
	}
}

func (producer *MolassesProducer) HandleBftMsg(bftMsg *quorumpb.BftMsg) error {
	molaproducer_log.Debugf("<%s> HandleBFTMsg called", producer.groupId)

	if bftMsg.Type != quorumpb.BftMsgType_HB_BFT {
		//unmarshal bft msg
		hbMsg := &quorumpb.HBMsgv1{}
		err := proto.Unmarshal(bftMsg.Data, hbMsg)
		if err != nil {
			molaproducer_log.Errorf("unmarshal bft msg failed with error: %s", err.Error())
			return err
		}

		if producer.ptbft != nil {
			producer.ptbft.HandleHBMessage(hbMsg)
		}
	}

	return nil
}

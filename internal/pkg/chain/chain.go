package chain

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	pubsubconn "github.com/huo-ju/quorum/internal/pkg/pubsubconn"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/protobuf/proto"

	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
)

var chain_log = logging.Logger("chain")

type GroupProducer struct {
	ProducerPubkey   string
	ProducerPriority int8
}

type Chain struct {
	nodename          string
	group             *Group
	userChannelId     string
	producerChannelId string
	trxMgrs           map[string]*TrxMgr
	ProducerPool      map[string]*quorumpb.ProducerItem

	Syncer    *Syncer
	Consensus Consensus
	statusmu  sync.RWMutex
}

func (chain *Chain) CustomInit(nodename string, group *Group, producerPubsubconn pubsubconn.PubSubConn, userPubsubconn pubsubconn.PubSubConn) {
	chain.group = group
	chain.trxMgrs = make(map[string]*TrxMgr)
	chain.nodename = nodename

	chain.producerChannelId = PRODUCER_CHANNEL_PREFIX + group.Item.GroupId
	producerTrxMgr := &TrxMgr{}
	producerTrxMgr.Init(chain.group.Item, producerPubsubconn)
	producerTrxMgr.SetNodeName(nodename)
	chain.trxMgrs[chain.producerChannelId] = producerTrxMgr

	chain.Consensus = NewMolasses(&MolassesProducer{}, &MolassesUser{})
	chain.Consensus.Producer().Init(group, chain.trxMgrs, chain.nodename)
	chain.Consensus.User().Init(group.Item, group.ChainCtx.nodename, chain)

	chain.userChannelId = USER_CHANNEL_PREFIX + group.Item.GroupId
	userTrxMgr := &TrxMgr{}
	userTrxMgr.Init(chain.group.Item, userPubsubconn)
	userTrxMgr.SetNodeName(nodename)
	chain.trxMgrs[chain.userChannelId] = userTrxMgr

	chain.Syncer = &Syncer{nodeName: nodename}
	chain.Syncer.Init(chain.group, producerTrxMgr)
}

func (chain *Chain) Init(group *Group) error {
	chain_log.Infof("Init called")
	chain.group = group
	chain.trxMgrs = make(map[string]*TrxMgr)
	chain.nodename = nodectx.GetNodeCtx().Name

	//create user channel
	chain.userChannelId = USER_CHANNEL_PREFIX + group.Item.GroupId
	chain.producerChannelId = PRODUCER_CHANNEL_PREFIX + group.Item.GroupId

	producerPsconn := pubsubconn.InitP2pPubSubConn(nodectx.GetNodeCtx().Ctx, nodectx.GetNodeCtx().Node.Pubsub, nodectx.GetNodeCtx().Name)
	producerPsconn.JoinChannel(chain.producerChannelId, chain)

	userPsconn := pubsubconn.InitP2pPubSubConn(nodectx.GetNodeCtx().Ctx, nodectx.GetNodeCtx().Node.Pubsub, nodectx.GetNodeCtx().Name)
	userPsconn.JoinChannel(chain.userChannelId, chain)

	//create user trx manager
	var userTrxMgr *TrxMgr
	userTrxMgr = &TrxMgr{}
	userTrxMgr.Init(chain.group.Item, userPsconn)
	chain.trxMgrs[chain.userChannelId] = userTrxMgr

	var producerTrxMgr *TrxMgr
	producerTrxMgr = &TrxMgr{}
	producerTrxMgr.Init(chain.group.Item, producerPsconn)
	chain.trxMgrs[chain.producerChannelId] = producerTrxMgr

	chain.Syncer = &Syncer{nodeName: chain.nodename}
	chain.Syncer.Init(chain.group, producerTrxMgr)

	return nil
}

func (chain *Chain) LoadProducer() {
	//load producer list
	chain.UpdProducerList()

	//check if need to create molass producer
	chain.UpdProducer()
}

func (chain *Chain) StartInitialSync(block *quorumpb.Block) error {
	chain_log.Infof("StartSync called")
	if chain.Syncer != nil {
		return chain.Syncer.SyncForward(block)
	}
	return nil
}

func (chain *Chain) StopSync() error {
	chain_log.Infof("StopSync called")
	if chain.Syncer != nil {
		return chain.Syncer.StopSync()
	}
	return nil
}

func (chain *Chain) GetProducerTrxMgr() *TrxMgr {
	return chain.trxMgrs[chain.producerChannelId]
}

func (chain *Chain) GetUserTrxMgr() *TrxMgr {
	return chain.trxMgrs[chain.userChannelId]
}

func (chain *Chain) UpdChainInfo(height int64, blockId []string) error {
	//update group info
	chain.group.Item.HighestHeight = height
	chain.group.Item.HighestBlockId = blockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	return nodectx.GetDbMgr().UpdGroup(chain.group.Item)
}

func (chain *Chain) HandleTrx(trx *quorumpb.Trx) error {
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Errorf("Trx Version mismatch %s", trx.TrxId)
		return errors.New("Trx Version mismatch")
	}
	switch trx.Type {
	case quorumpb.TrxType_AUTH:
		chain.handleTrx(trx)
	case quorumpb.TrxType_POST:
		chain.handleTrx(trx)
	case quorumpb.TrxType_ANNOUNCE:
		chain.handleTrx(trx)
	case quorumpb.TrxType_PRODUCER:
		chain.handleTrx(trx)
	case quorumpb.TrxType_SCHEMA:
		chain.handleTrx(trx)
	case quorumpb.TrxType_REQ_BLOCK_FORWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockForward(trx)
	case quorumpb.TrxType_REQ_BLOCK_BACKWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockBackward(trx)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockResp(trx)
	case quorumpb.TrxType_BLOCK_PRODUCED:
		chain.handleBlockProduced(trx)
		return nil
	default:
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func (chain *Chain) HandleBlock(block *quorumpb.Block) error {
	chain_log.Infof("HandleBlock called")

	var shouldAccept bool

	// check if block produced by registed producer or group owner
	if len(chain.ProducerPool) == 0 && block.ProducerPubKey == chain.group.Item.OwnerPubKey {
		//from owner, no registed producer
		shouldAccept = true
	} else if _, ok := chain.ProducerPool[block.ProducerPubKey]; ok {
		//from registed producer
		shouldAccept = true
	} else {
		//from someone else
		shouldAccept = false
		chain_log.Warnf("received block from unknow node with producer key %s", block.ProducerPubKey)
	}

	if shouldAccept {
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Infof(err.Error())
		}
	}

	return nil
}

func (chain *Chain) handleTrx(trx *quorumpb.Trx) error {
	chain_log.Infof("handleTrx called")
	//if I am not a producer, do nothing
	if chain.Consensus == nil {
		return nil
	}
	chain.Consensus.Producer().AddTrx(trx)
	return nil
}

func (chain *Chain) handleReqBlockForward(trx *quorumpb.Trx) error {
	chain_log.Infof("handleReqBlockForward called")
	if chain.Consensus == nil {
		return nil
	}
	return chain.Consensus.Producer().GetBlockForward(trx)
}

func (chain *Chain) handleReqBlockBackward(trx *quorumpb.Trx) error {
	chain_log.Infof("handleReqBlockBackward called")
	if chain.Consensus == nil {
		return nil
	}
	return chain.Consensus.Producer().GetBlockBackward(trx)
}

func (chain *Chain) handleReqBlockResp(trx *quorumpb.Trx) error {
	chain_log.Infof("handleReqBlockResp called")

	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
	if err != nil {
		return err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return err
	}

	var reqBlockResp quorumpb.ReqBlockResp
	if err := proto.Unmarshal(decryptData, &reqBlockResp); err != nil {
		return err
	}

	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		chain_log.Infof("Not asked by me, ignore")
		return nil
	}

	var newBlock quorumpb.Block
	if err := proto.Unmarshal(reqBlockResp.Block, &newBlock); err != nil {
		return err
	}

	var shouldAccept bool

	chain_log.Debugf("block producer %s, block_id %s", newBlock.ProducerPubKey, newBlock.BlockId)

	if _, ok := chain.ProducerPool[newBlock.ProducerPubKey]; ok {
		shouldAccept = true
	} else {
		shouldAccept = false
	}

	if !shouldAccept {
		chain_log.Warnf("Block not produced by registed producer, reject it")
		return nil
	}

	return chain.Syncer.AddBlockSynced(&reqBlockResp, &newBlock)
}

func (chain *Chain) handleBlockProduced(trx *quorumpb.Trx) error {
	chain_log.Infof("handleBlockProduced called")
	if chain.Consensus == nil {
		return nil
	}
	return chain.Consensus.Producer().AddProducedBlock(trx)
}

func (chain *Chain) UpdProducerList() {
	//create and load group producer pool
	chain.ProducerPool = make(map[string]*quorumpb.ProducerItem)
	producers, _ := nodectx.GetDbMgr().GetProducers(chain.group.Item.GroupId, chain.nodename)
	for _, item := range producers {
		chain.ProducerPool[item.ProducerPubkey] = item
		ownerPrefix := ""
		if item.ProducerPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("Load producer %s %s", item.ProducerPubkey, ownerPrefix)
	}
}

func (chain *Chain) UpdProducer() {
	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
		//Yes, I am producer, create censensus and group producer
		chain_log.Infof("Create and initial molasses producer")
		chain.Consensus = NewMolasses(&MolassesProducer{NodeName: chain.nodename}, &MolassesUser{NodeName: chain.nodename})
		chain.Consensus.Producer().Init(chain.group, chain.trxMgrs, chain.nodename)
		chain.Consensus.User().Init(chain.group)
	} else {
		chain_log.Infof("Set molasses producer to nil")
		chain.Consensus = nil
	}
}

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

	groupId string
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
	chain.Consensus.Producer().Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
	chain.Consensus.User().Init(group.Item, group.ChainCtx.nodename, chain)

	chain.userChannelId = USER_CHANNEL_PREFIX + group.Item.GroupId
	userTrxMgr := &TrxMgr{}
	userTrxMgr.Init(chain.group.Item, userPubsubconn)
	userTrxMgr.SetNodeName(nodename)
	chain.trxMgrs[chain.userChannelId] = userTrxMgr

	chain.Syncer = &Syncer{nodeName: nodename}
	chain.Syncer.Init(chain.group, producerTrxMgr)

	chain.groupId = group.Item.GroupId
}

func (chain *Chain) Init(group *Group) error {
	chain_log.Debugf("<%s> Init called", group.Item.GroupId)
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

	chain.groupId = group.Item.GroupId

	chain_log.Infof("<%s> chainctx initialed", chain.groupId)
	return nil
}

func (chain *Chain) StartInitialSync(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> StartInitialSync called", chain.groupId)
	if chain.Syncer != nil {
		return chain.Syncer.SyncForward(block)
	}
	return nil
}

func (chain *Chain) StopSync() error {
	chain_log.Debugf("<%s> StopSync called", chain.groupId)
	if chain.Syncer != nil {
		return chain.Syncer.StopSync()
	}
	return nil
}

func (chain *Chain) GetProducerTrxMgr() *TrxMgr {
	chain_log.Debugf("<%s> GetProducerTrxMgr called", chain.groupId)
	return chain.trxMgrs[chain.producerChannelId]
}

func (chain *Chain) GetUserTrxMgr() *TrxMgr {
	chain_log.Debugf("<%s> GetUserTrxMgr called", chain.groupId)
	return chain.trxMgrs[chain.userChannelId]
}

func (chain *Chain) UpdChainInfo(height int64, blockId []string) error {
	chain_log.Debugf("<%s> UpdChainInfo called", chain.groupId)
	chain.group.Item.HighestHeight = height
	chain.group.Item.HighestBlockId = blockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	chain_log.Infof("<%s> Chain Info updated %d, %v", chain.group.Item.GroupId, height, blockId)
	return nodectx.GetDbMgr().UpdGroup(chain.group.Item)
}

func (chain *Chain) HandleTrx(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleTrx called", chain.groupId)
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Errorf("Trx Version mismatch %s", trx.TrxId)
		return errors.New("Trx Version mismatch")
	}
	switch trx.Type {
	case quorumpb.TrxType_AUTH:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_POST:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_ANNOUNCE:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_PRODUCER:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_SCHEMA:
		chain.producerAddTrx(trx)
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
		chain_log.Warningf("<%s> unsupported msg type", chain.group.Item.GroupId)
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func (chain *Chain) HandleBlock(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> HandleBlock called", chain.groupId)

	var shouldAccept bool

	// check if block produced by registed producer or group owner
	if len(chain.ProducerPool) == 1 && block.ProducerPubKey == chain.group.Item.OwnerPubKey {
		//from owner and no other registed producer
		shouldAccept = true
	} else if _, ok := chain.ProducerPool[block.ProducerPubKey]; ok {
		//from registed producer
		shouldAccept = true
	} else {
		//from someone else
		shouldAccept = false
		chain_log.Warningf("<%s> received block <%s> from unregisted producer <%s>, reject it", chain.group.Item.GroupId, block.BlockId, block.ProducerPubKey)
	}

	if shouldAccept {
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Debug(err.Error())
		}
	}

	return nil
}

func (chain *Chain) producerAddTrx(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> producerAddTrx called", chain.groupId)
	if chain.Consensus.Producer() == nil {
		return nil
	}
	chain.Consensus.Producer().AddTrx(trx)
	return nil
}

func (chain *Chain) handleReqBlockForward(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> handleReqBlockForward called", chain.groupId)
	if chain.Consensus.Producer() == nil {
		return nil
	}
	return chain.Consensus.Producer().GetBlockForward(trx)
}

func (chain *Chain) handleReqBlockBackward(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> handleReqBlockBackward called", chain.groupId)
	if chain.Consensus.Producer() == nil {
		return nil
	}
	return chain.Consensus.Producer().GetBlockBackward(trx)
}

func (chain *Chain) handleReqBlockResp(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupId)

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

	//if asked by myself, do nothing
	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		return nil
	}

	var newBlock quorumpb.Block
	if err := proto.Unmarshal(reqBlockResp.Block, &newBlock); err != nil {
		return err
	}

	var shouldAccept bool

	chain_log.Infof("<%s> REQ_BLOCK_RESP, block <%s> from producer <%s>", chain.groupId, newBlock.BlockId, newBlock.ProducerPubKey)

	if _, ok := chain.ProducerPool[newBlock.ProducerPubKey]; ok {
		shouldAccept = true
	} else {
		shouldAccept = false
	}

	if !shouldAccept {
		chain_log.Warnf(" <%s> Block producer <%s> not registed, reject", chain.groupId, newBlock.ProducerPubKey)
		return nil
	}

	return chain.Syncer.AddBlockSynced(&reqBlockResp, &newBlock)
}

func (chain *Chain) handleBlockProduced(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> handleBlockProduced called", chain.groupId)
	if chain.Consensus.Producer() == nil {
		return nil
	}
	return chain.Consensus.Producer().AddProducedBlock(trx)
}

func (chain *Chain) UpdProducerList() {
	chain_log.Debugf("<%s> UpdProducerList called", chain.groupId)
	//create and load group producer pool
	chain.ProducerPool = make(map[string]*quorumpb.ProducerItem)
	producers, _ := nodectx.GetDbMgr().GetProducers(chain.group.Item.GroupId, chain.nodename)
	for _, item := range producers {
		chain.ProducerPool[item.ProducerPubkey] = item
		ownerPrefix := ""
		if item.ProducerPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(group owner)"
		}
		chain_log.Infof("<%s> Load producer <%s%s>", chain.groupId, item.ProducerPubkey, ownerPrefix)
	}
}

func (chain *Chain) CreateConsensus() {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupId)
	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
		//Yes, I am producer, create group producer/user
		chain_log.Infof("<%s> Create and initial molasses producer/user", chain.groupId)
		chain.Consensus = NewMolasses(&MolassesProducer{}, &MolassesUser{})
		chain.Consensus.Producer().Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
		chain.Consensus.User().Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
	} else {
		chain_log.Infof("<%s> Create and initial molasses user", chain.groupId)
		chain.Consensus = NewMolasses(nil, &MolassesUser{})
		chain.Consensus.User().Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
	}
}

func (chain *Chain) IsSyncerReady() bool {
	chain_log.Debugf("<%s> IsSyncerReady called", chain.groupId)
	if chain.Syncer.Status == SYNCING_BACKWARD ||
		chain.Syncer.Status == SYNCING_FORWARD ||
		chain.Syncer.Status == SYNC_FAILED {
		chain_log.Debugf("<%s>group is syncing or sync failed", chain.groupId)
		return true
	}

	return false
}

package chain

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"

	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
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
	syncChannelId     string
	trxMgrs           map[string]*TrxMgr
	ProducerPool      map[string]*quorumpb.ProducerItem
	userPool          map[string]*quorumpb.UserItem
	peerIdPool        map[string]string

	Syncer    *Syncer
	Consensus Consensus
	statusmu  sync.RWMutex

	producerChannTimer *time.Timer
	groupId            string

	ProviderPeerIdPool map[string]string
}

func (chain *Chain) Init(group *Group) error {
	chain_log.Debugf("<%s> Init called", group.Item.GroupId)
	chain.group = group
	chain.trxMgrs = make(map[string]*TrxMgr)
	chain.nodename = nodectx.GetNodeCtx().Name
	chain.groupId = group.Item.GroupId

	chain.producerChannelId = PRODUCER_CHANNEL_PREFIX + chain.groupId
	chain.userChannelId = USER_CHANNEL_PREFIX + chain.groupId
	chain.syncChannelId = SYNC_CHANNEL_PREFIX + chain.groupId + "_" + chain.group.Item.UserSignPubkey

	chain.ProviderPeerIdPool = make(map[string]string)

	chain_log.Infof("<%s> chainctx initialed", chain.groupId)
	return nil
}

func (chain *Chain) LeaveChannel() error {
	chain_log.Debugf("<%s> LeaveChannel called", chain.groupId)
	if _, ok := chain.trxMgrs[chain.userChannelId]; ok {
		nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(chain.userChannelId)
		delete(chain.trxMgrs, chain.userChannelId)

	}
	if _, ok := chain.trxMgrs[chain.producerChannelId]; ok {
		nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(chain.producerChannelId)
		delete(chain.trxMgrs, chain.producerChannelId)
	}
	if _, ok := chain.trxMgrs[chain.syncChannelId]; ok {
		nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(chain.syncChannelId)
		delete(chain.trxMgrs, chain.syncChannelId)
	}

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

func (chain *Chain) GetChainCtx() *Chain {
	return chain
}

func (chain *Chain) GetProducerTrxMgr() *TrxMgr {
	chain_log.Debugf("<%s> GetProducerTrxMgr called", chain.groupId)

	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
		return chain.trxMgrs[chain.producerChannelId]
	}

	var producerTrxMgr *TrxMgr

	if _, ok := chain.trxMgrs[chain.producerChannelId]; ok {
		producerTrxMgr = chain.trxMgrs[chain.producerChannelId]
		chain_log.Debugf("<%s> reset connection timer for producertrxMgr <%s>", chain.groupId, chain.producerChannelId)
		chain.producerChannTimer.Stop()
		chain.producerChannTimer.Reset(CLOSE_CONN_TIMER * time.Second)

	} else {
		chain.createProducerTrxMgr()
		producerTrxMgr = chain.trxMgrs[chain.producerChannelId]
		chain_log.Debugf("<%s> create close_conn timer for producer channel <%s>", chain.groupId, chain.producerChannelId)
		chain.producerChannTimer = time.AfterFunc(CLOSE_CONN_TIMER*time.Second, func() {
			if _, ok := chain.trxMgrs[chain.producerChannelId]; ok {
				chain_log.Debugf("<%s> time up, close sync channel <%s>", chain.groupId, chain.producerChannelId)
				nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(chain.producerChannelId)
				delete(chain.trxMgrs, chain.producerChannelId)
			}
		})
	}

	return producerTrxMgr
}

func (chain *Chain) GetUserTrxMgr() *TrxMgr {
	chain_log.Debugf("<%s> GetUserTrxMgr called", chain.groupId)
	return chain.trxMgrs[chain.userChannelId]
}

func (chain *Chain) UpdChainInfo(height int64, blockId string) error {
	chain_log.Debugf("<%s> UpdChainInfo called", chain.groupId)
	chain.group.Item.HighestHeight = height
	chain.group.Item.HighestBlockId = blockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	chain_log.Infof("<%s> Chain Info updated %d, %v", chain.group.Item.GroupId, height, blockId)
	return nodectx.GetDbMgr().UpdGroup(chain.group.Item)
}

func (chain *Chain) HandleTrx(trx *quorumpb.Trx) error {
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Errorf("HandleTrx called, Trx Version mismatch %s: %s vs %s", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return errors.New("Trx Version mismatch")
	}
	switch trx.Type {
	case quorumpb.TrxType_POST:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_ANNOUNCE:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_PRODUCER:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_USER:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_SCHEMA:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_APP_CONFIG:
		chain.producerAddTrx(trx)
	case quorumpb.TrxType_CHAIN_CONFIG:
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
	//case quorumpb.TrxType_ASK_PEERID:
	//	chain.HandleAskPeerID(trx)
	//	return nil
	//case quorumpb.TrxType_ASK_PEERID_RESP:
	//	chain.HandleAskPeerIdResp(trx)
	//	return nil
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

	if chain.Consensus.Producer() != nil {
		//if I am a producer, no need to addBlock since block just produced is already saved
		shouldAccept = false
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
			chain_log.Debugf("<%s> user add block error <%s>", chain.groupId, err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				chain_log.Infof("<%s>, parent not exist, sync backward from block <%s>", chain.groupId, block.BlockId)
				chain.Syncer.SyncBackward(block)
			}
		}
	}

	return nil
}

func (chain *Chain) producerAddTrx(trx *quorumpb.Trx) error {
	if chain.Consensus != nil && chain.Consensus.Producer() == nil {
		return nil
	}
	chain_log.Debugf("<%s> producerAddTrx called", chain.groupId)
	chain.Consensus.Producer().AddTrx(trx)
	return nil
}

func (chain *Chain) handleReqBlockForward(trx *quorumpb.Trx) error {
	if chain.Consensus != nil && chain.Consensus.Producer() == nil {
		return nil
	}
	chain_log.Debugf("<%s> producer handleReqBlockForward called", chain.groupId)
	return chain.Consensus.Producer().GetBlockForward(trx)
}

func (chain *Chain) handleReqBlockBackward(trx *quorumpb.Trx) error {
	if chain.Consensus != nil && chain.Consensus.Producer() == nil {
		return nil
	}
	chain_log.Debugf("<%s> producer handleReqBlockBackward called", chain.groupId)
	return chain.Consensus.Producer().GetBlockBackward(trx)
}

func (chain *Chain) handleReqBlockResp(trx *quorumpb.Trx) error {
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

	//if not asked by myself, ignore it
	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		return nil
	}

	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupId)

	var newBlock quorumpb.Block
	if err := proto.Unmarshal(reqBlockResp.Block, &newBlock); err != nil {
		return err
	}

	var shouldAccept bool

	chain_log.Debugf("<%s> REQ_BLOCK_RESP, block_id <%s>, block_producer <%s>", chain.groupId, newBlock.BlockId, newBlock.ProducerPubKey)

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
	if chain.Consensus != nil && chain.Consensus.Producer() == nil {
		return nil
	}
	chain_log.Debugf("<%s> handleBlockProduced called", chain.groupId)
	return chain.Consensus.Producer().AddProducedBlock(trx)
}

func (chain *Chain) UpdProducerList() {
	chain_log.Debugf("<%s> UpdProducerList called", chain.groupId)
	//create and load group producer pool
	chain.ProducerPool = make(map[string]*quorumpb.ProducerItem)
	producers, _ := nodectx.GetDbMgr().GetProducers(chain.group.Item.GroupId, chain.nodename)
	for _, item := range producers {
		chain.ProducerPool[item.ProducerPubkey] = item
		ownerPrefix := "(producer)"
		if item.ProducerPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> Load producer <%s%s>", chain.groupId, item.ProducerPubkey, ownerPrefix)
	}

	//update announced producer result
	announcedProducers, _ := nodectx.GetDbMgr().GetAnnounceProducersByGroup(chain.group.Item.GroupId, chain.nodename)
	for _, item := range announcedProducers {
		_, ok := chain.ProducerPool[item.SignPubkey]
		err := nodectx.GetDbMgr().UpdateAnnounceResult(quorumpb.AnnounceType_AS_PRODUCER, chain.group.Item.GroupId, item.SignPubkey, ok, chain.nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.groupId, err.Error())
		}
	}
}

func (chain *Chain) HandleAskPeerID(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleAskPeerID called", chain.groupId)
	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return nil
	}
	return chain.Consensus.Producer().HandleAskPeerId(trx)
}

//func (chain *Chain) HandleAskPeerIdResp(trx *quorumpb.Trx) error {
//	chain_log.Debugf("<%s> HandleAskPeerIdResp called", chain.groupId)
//
//	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
//	if err != nil {
//		return err
//	}
//
//	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
//	if err != nil {
//		return err
//	}
//
//	var respItem quorumpb.AskPeerIdResp
//
//	if err := proto.Unmarshal(decryptData, &respItem); err != nil {
//		return err
//	}
//
//	//update peerId table
//	chain.ProviderPeerIdPool[respItem.RespPeerPubkey] = respItem.RespPeerId
//	chain_log.Debugf("<%s> Pubkey<%s> PeerId<%s> ", chain.groupId, respItem.RespPeerPubkey, &respItem.RespPeerId)
//	//initial both producerChannel and syncChannel
//	//err = chain.InitSession(chain.producerChannelId)
//	//if err != nil {
//	//	return err
//	//}
//
//	//err = chain.InitSession(chain.syncChannelId)
//	//if err != nil {
//	//	return err
//	//}
//
//	return nil
//}

func (chain *Chain) GetUserPool() map[string]*quorumpb.UserItem {
	return chain.userPool
}

func (chain *Chain) GetUsesEncryptPubKeys() ([]string, error) {
	keys := []string{}
	ks := nodectx.GetNodeCtx().Keystore
	mypubkey, err := ks.GetEncodedPubkey(chain.group.Item.GroupId, localcrypto.Encrypt)
	if err != nil {
		return nil, err
	}
	keys = append(keys, mypubkey)
	for _, usr := range chain.userPool {
		if usr.EncryptPubkey != mypubkey {
			keys = append(keys, usr.EncryptPubkey)
		}
	}

	return keys, nil
}

func (chain *Chain) UpdUserList() {
	chain_log.Debugf("<%s> UpdUserList called", chain.groupId)
	//create and load group user pool
	chain.userPool = make(map[string]*quorumpb.UserItem)
	users, _ := nodectx.GetDbMgr().GetUsers(chain.group.Item.GroupId, chain.nodename)
	for _, item := range users {
		chain.userPool[item.UserPubkey] = item
		ownerPrefix := "(user)"
		if item.UserPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> Load Users <%s%s>", chain.groupId, item.UserPubkey, ownerPrefix)
	}

	//update announced User result
	announcedUsers, _ := nodectx.GetDbMgr().GetAnnounceUsersByGroup(chain.group.Item.GroupId, chain.nodename)
	for _, item := range announcedUsers {
		_, ok := chain.userPool[item.SignPubkey]
		err := nodectx.GetDbMgr().UpdateAnnounceResult(quorumpb.AnnounceType_AS_USER, chain.group.Item.GroupId, item.SignPubkey, ok, chain.nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.groupId, err.Error())
		}
	}
}

func (chain *Chain) CreateConsensus() {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupId)

	var user User
	var producer Producer

	if chain.Consensus == nil || chain.Consensus.User() == nil {
		chain_log.Infof("<%s> Create and initial molasses user", chain.groupId)
		user = &MolassesUser{}
		user.Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
	} else {
		chain_log.Infof("<%s> reuse molasses user", chain.groupId)
		user = chain.Consensus.User()
	}

	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
		if chain.Consensus == nil || chain.Consensus.Producer() == nil {
			chain_log.Infof("<%s> Create and initial molasses producer", chain.groupId)
			producer = &MolassesProducer{}
			producer.Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
			chain.createProducerTrxMgr()
		} else {
			chain_log.Infof("<%s> reuse molasses producer", chain.groupId)
			producer = chain.Consensus.Producer()
		}
	} else {
		chain_log.Infof("<%s> no producer created", chain.groupId)
		producer = nil
	}

	if chain.Consensus == nil {
		chain_log.Infof("<%s> created consensus", chain.groupId)
		chain.Consensus = NewMolasses(producer, user)
	} else {
		chain_log.Infof("<%s> reuse consensus", chain.groupId)
		chain.Consensus.SetProducer(producer)
		chain.Consensus.SetUser(user)
	}

	chain.createUserTrxMgr()
	chain.createSyncTrxMgr()

	if chain.Syncer == nil {
		chain_log.Infof("<%s> Create and init group syncer", chain.groupId)
		chain.Syncer = &Syncer{nodeName: chain.nodename}
		chain.Syncer.Init(chain.group, chain)
	} else {
		chain_log.Infof("<%s> reuse syncer", chain.groupId)
	}

	//_ = chain.InitSession(chain.producerChannelId)
	//_ = chain.InitSession(chain.syncChannelId)
}

func (chain *Chain) createUserTrxMgr() {
	chain_log.Infof("<%s> Create and join group user channel", chain.groupId)

	if _, ok := chain.trxMgrs[chain.userChannelId]; ok {
		chain_log.Infof("<%s> reuse user channel", chain.groupId)
		return
	}
	userPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(chain.userChannelId, chain)
	chain_log.Infof("<%s> Create and init group userTrxMgr", chain.groupId)
	var userTrxMgr *TrxMgr
	userTrxMgr = &TrxMgr{}
	userTrxMgr.Init(chain.group.Item, userPsconn)
	chain.trxMgrs[chain.userChannelId] = userTrxMgr
}

func (chain *Chain) createSyncTrxMgr() {
	chain_log.Infof("<%s> Create and join group syncer channel", chain.groupId)

	if _, ok := chain.trxMgrs[chain.syncChannelId]; ok {
		chain_log.Infof("<%s> reuse syncer channel", chain.groupId)
		return
	}

	syncPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(chain.syncChannelId, chain)
	chain_log.Infof("<%s> Create and init group syncTrxMgr", chain.groupId)
	var syncTrxMgr *TrxMgr
	syncTrxMgr = &TrxMgr{}
	syncTrxMgr.Init(chain.group.Item, syncPsconn)
	chain.trxMgrs[chain.syncChannelId] = syncTrxMgr
}

func (chain *Chain) createProducerTrxMgr() {
	chain_log.Infof("<%s> Create and join group producer channel", chain.groupId)
	if _, ok := chain.trxMgrs[chain.producerChannelId]; ok {
		chain_log.Infof("<%s> reuse producer channel", chain.groupId)
		return
	}

	producerPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(chain.producerChannelId, chain)
	chain_log.Infof("<%s> Create and init group producerTrxMgr", chain.groupId)
	var producerTrxMgr *TrxMgr
	producerTrxMgr = &TrxMgr{}
	producerTrxMgr.Init(chain.group.Item, producerPsconn)
	chain.trxMgrs[chain.producerChannelId] = producerTrxMgr
}

func (chain *Chain) InitSession(channelId string) error {
	return nil
	//	chain_log.Debugf("<%s> InitSession called", chain.groupId)
	//	//err := nodectx.GetNodeCtx().Node.RumExchange.ConnectRex(nodectx.GetNodeCtx().Ctx)
	//	//if err != nil {
	//	//	return err
	//	//}
	//	if peerId, ok := chain.ProviderPeerIdPool[chain.group.Item.OwnerPubKey]; ok {
	//		return nodectx.GetNodeCtx().Node.RumExchange.InitSession(peerId, channelId)
	//	} else {
	//		return chain.AskPeerId()
	//	}
}

func (chain *Chain) AskPeerId() error {
	chain_log.Debugf("<%s> AskPeerId called", chain.groupId)
	var req quorumpb.AskPeerId
	req = quorumpb.AskPeerId{}

	req.GroupId = chain.groupId
	req.UserPeerId = nodectx.GetNodeCtx().Node.PeerID.Pretty()

	return chain.GetProducerTrxMgr().SendAskPeerId(&req)
}
func (chain *Chain) IsSyncerReady() bool {
	chain_log.Debugf("<%s> IsSyncerReady called", chain.groupId)
	if chain.Syncer.Status == SYNCING_BACKWARD ||
		chain.Syncer.Status == SYNCING_FORWARD ||
		chain.Syncer.Status == SYNC_FAILED {
		chain_log.Debugf("<%s> syncer is busy, status: <%d>", chain.groupId, chain.Syncer.Status)
		return true
	}
	chain_log.Debugf("<%s> syncer is IDLE", chain.groupId)
	return false
}

func (chain *Chain) SyncBackward(block *quorumpb.Block) error {
	return chain.Syncer.SyncBackward(block)
}

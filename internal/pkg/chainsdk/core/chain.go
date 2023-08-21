package chain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/consensus"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
)

var chain_log = logging.Logger("chain")

var DEFAULT_PROPOSE_TRX_INTERVAL = 1000 //ms

type Chain struct {
	groupItem     *quorumpb.GroupItem
	nodename      string
	producerPool  map[string]*quorumpb.ProducerItem
	syncerPool    map[string]*quorumpb.Syncer
	trxFactory    *rumchaindata.TrxFactory
	rexSyncer     *RexLiteSyncer
	chaindata     *ChainData
	Consensus     def.Consensus
	CurrBlock     uint64
	CurrEpoch     uint64
	LatestUpdate  int64
	ChainCtx      context.Context
	CtxCancelFunc context.CancelFunc
}

func (chain *Chain) NewChainWithSeed(seed *quorumpb.GroupSeed, item *quorumpb.GroupItem, nodename string) error {
	chain_log.Debugf("<%s> NewChain called", item.GroupId)

	var forkItem *quorumpb.ForkItem

	if seed.GenesisBlock.Consensus.Type == quorumpb.GroupConsenseType_POA {
		poaConsensus := &quorumpb.PoaConsensusInfo{}
		err := proto.Unmarshal(seed.GenesisBlock.Consensus.Data, poaConsensus)
		if err != nil {
			chain_log.Debugf("<%s> Unmarshal failed with error <%s>", chain.groupItem.GroupId, err.Error())
			return err
		}
		forkItem = poaConsensus.ForkInfo

	} else {
		chain_log.Warnf("<%s> unsupported consensus type <%s>", chain.groupItem.GroupId, seed.GenesisBlock.Consensus.Type.String())
		return errors.New("unsupported consensus type")
	}

	chain.groupItem = item
	chain.nodename = nodename

	//initial TrxFactory
	chain.trxFactory = &rumchaindata.TrxFactory{}
	chain.trxFactory.Init(nodectx.GetNodeCtx().Version, chain.groupItem, chain.nodename)

	//create context with cancel function, chainCtx will be ctx parent of all underlay components
	chain.ChainCtx, chain.CtxCancelFunc = context.WithCancel(nodectx.GetNodeCtx().Ctx)

	//initial chaindata manager
	chain.chaindata = &ChainData{
		nodename:       chain.nodename,
		groupId:        chain.groupItem.GroupId,
		groupCipherKey: chain.groupItem.CipherKey,
		userSignPubkey: chain.groupItem.UserSignPubkey,
		dbmgr:          nodectx.GetDbMgr(),
	}

	chain_log.Debugf("<%s> NewChain done", chain.groupItem.GroupId)
	//initial Syncer
	// chain.rexSyncer = NewRexSyncer(chain.ChainCtx, chain.groupItem, chain.nodename, chain, chain)
	chain.rexSyncer = NewRexLiteSyncer(chain.ChainCtx, chain.groupItem, chain.nodename, chain, chain)

	chain_log.Debugf("<%s> initial chain config", item.GroupId)
	currEpoch := forkItem.EpochDuration
	currBlockId := forkItem.StartFromBlock
	lastUpdate := time.Now().UnixNano()
	chain.SetCurrEpoch(currEpoch)
	chain.SetCurrBlockId(currBlockId)
	chain.SetLastUpdate(lastUpdate)
	chain_log.Debugf("<%s> CurrEpoch <%d> CurrBlockId <%d> lastUpdate <%d>", chain.groupItem.GroupId, currEpoch, currBlockId, lastUpdate)
	chain.SaveChainInfoToDb()

	return nil
}

func (chain *Chain) NewChain(item *quorumpb.GroupItem, nodename string, loadChainInfo bool) error {
	chain_log.Debugf("<%s> NewChain called, commented by cuicat", item.GroupId)

	/*
		chain.groupItem = item
		chain.nodename = nodename

		//initial TrxFactory
		chain.trxFactory = &rumchaindata.TrxFactory{}
		chain.trxFactory.Init(nodectx.GetNodeCtx().Version, chain.groupItem, chain.nodename)

		//create context with cancel function, chainCtx will be ctx parent of all underlay components
		chain.ChainCtx, chain.CtxCancelFunc = context.WithCancel(nodectx.GetNodeCtx().Ctx)

		//initial Syncer
		chain.rexSyncer = NewRexLiteSyncer(chain.ChainCtx, chain.groupItem, chain.nodename, chain, chain)

		//initial chaindata manager
		chain.chaindata = &ChainData{
			nodename:       chain.nodename,
			groupId:        chain.groupItem.GroupId,
			groupCipherKey: chain.groupItem.CipherKey,
			userSignPubkey: chain.groupItem.UserSignPubkey,
			dbmgr:          nodectx.GetDbMgr()}

		if loadChainInfo {
			chain_log.Debugf("<%s> load chain config", item.GroupId)
			currBlockId, currEpoch, lastUpdate, err := nodectx.GetNodeCtx().GetChainStorage().GetChainInfo(chain.groupItem.GroupId, chain.nodename)
			if err != nil {
				return err
			}
			chain.SetCurrEpoch(currEpoch)
			chain.SetLastUpdate(lastUpdate)
			chain.SetCurrBlockId(currBlockId)
			chain_log.Debugf("<%s> CurrEpoch <%d> CurrBlockId <%d> lastUpdate <%d>", chain.groupItem.GroupId, currEpoch, currBlockId, lastUpdate)
		} else {
			chain_log.Debugf("<%s> initial chain config", item.GroupId)
			currEpoch := uint64(0)
			currBlockId := uint64(0)
			lastUpdate := time.Now().UnixNano()
			chain.SetCurrEpoch(currEpoch)
			chain.SetCurrBlockId(currBlockId)
			chain.SetLastUpdate(lastUpdate)
			chain_log.Debugf("<%s> CurrEpoch <%d> CurrBlockId <%d> lastUpdate <%d>", chain.groupItem.GroupId, currEpoch, currBlockId, lastUpdate)
			chain.SaveChainInfoToDb()

			//initial consensus
			chain_log.Debugf("<%s> initial consensus", item.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().SetProducerConsensusConfInterval(chain.groupItem.GroupId, uint64(DEFAULT_PROPOSE_TRX_INTERVAL), chain.nodename)
		}

		chain_log.Debugf("<%s> NewChain done", chain.groupItem.GroupId)
		//initial Syncer
		// chain.rexSyncer = NewRexSyncer(chain.ChainCtx, chain.groupItem, chain.nodename, chain, chain)
		chain.rexSyncer = NewRexLiteSyncer(chain.ChainCtx, chain.groupItem, chain.nodename, chain, chain)
	*/
	return nil
}

// atomic opt for currEpoch
func (chain *Chain) SetCurrEpoch(currEpoch uint64) {
	atomic.StoreUint64(&chain.CurrEpoch, currEpoch)
}

func (chain *Chain) IncCurrEpoch() {
	atomic.AddUint64(&chain.CurrEpoch, 1)
}

func (chain *Chain) GetCurrEpoch() uint64 {
	return atomic.LoadUint64(&chain.CurrEpoch)
}

// atomic opt for currBlock
func (chain *Chain) SetCurrBlockId(currBlock uint64) {
	atomic.StoreUint64(&chain.CurrBlock, currBlock)
}

func (chain *Chain) IncCurrBlockId() {
	atomic.AddUint64(&chain.CurrBlock, 1)
}

func (chain *Chain) GetCurrBlockId() uint64 {
	return atomic.LoadUint64(&chain.CurrBlock)
}

// atomic opt for lastUpdate
func (chain *Chain) SetLastUpdate(lastUpdate int64) {
	atomic.StoreInt64(&chain.LatestUpdate, lastUpdate)
}

func (chain *Chain) GetLastUpdate() int64 {
	return atomic.LoadInt64(&chain.LatestUpdate)
}

func (chain *Chain) SaveChainInfoToDb() error {
	//chain_log.Debugf("<%s> SaveChainInfoToDb called", chain.groupItem.GroupId)
	//chain_log.Debugf("<%s> CurrEpoch <%d> CurrBlockId <%d> lastUpdate <%d>", chain.groupItem.GroupId, chain.GetCurrEpoch(), chain.GetCurrBlockId(), chain.GetLastUpdate())
	return nodectx.GetNodeCtx().GetChainStorage().SaveChainInfo(chain.GetCurrBlockId(), chain.GetCurrEpoch(), chain.GetLastUpdate(), chain.groupItem.GroupId, chain.nodename)
}

func (chain *Chain) GetTrxFactory() chaindef.TrxFactoryIface {
	chain_log.Debugf("<%s> GetTrxFactory called", chain.groupItem.GroupId)
	return chain.trxFactory
}

// PSConn msg handler
func (chain *Chain) HandlePsConnMessage(pkg *quorumpb.Package) error {
	//chain_log.Debugf("<%s> HandlePsConnMessage called, <%s>", chain.groupItem.GroupId, pkg.Type.String())
	var err error
	if pkg.Type == quorumpb.PackageType_BLOCK {
		blk := &quorumpb.Block{}
		err = proto.Unmarshal(pkg.Data, blk)
		if err != nil {
			chain_log.Warning(err.Error())
		} else {
			//TODO: save to cache, waitting for syncer to pickup it
			nodectx.GetNodeCtx().GetChainStorage().AddBlockToDSCache(blk, chain.nodename)
			chain.rexSyncer.TaskTrigger()
			// err = chain.HandleBlockPsConn(blk)
		}

	} else if pkg.Type == quorumpb.PackageType_TRX {
		trx := &quorumpb.Trx{}
		err = proto.Unmarshal(pkg.Data, trx)
		if err != nil {
			chain_log.Warningf(err.Error())
		} else {
			err = chain.HandleTrxPsConn(trx)
		}
	} else if pkg.Type == quorumpb.PackageType_BFT_MSG {
		bftMsg := &quorumpb.BftMsg{}
		err = proto.Unmarshal(pkg.Data, bftMsg)
		if err != nil {
			chain_log.Warningf(err.Error())
		} else {
			err = chain.HandleBftMsgPsConn(bftMsg)
		}
	} else if pkg.Type == quorumpb.PackageType_BROADCAST_MSG {
		broadcastMsg := &quorumpb.BroadcastMsg{}
		err = proto.Unmarshal(pkg.Data, broadcastMsg)
		if err != nil {
			chain_log.Warnf(err.Error())
		} else {
			err = chain.HandleBroadcastMsgPsConn(broadcastMsg)
		}
	} else {
		chain_log.Warningf("invalid pkg type <%s> for psconn", pkg.Type.String())
	}

	return err
}

// Handle Trx from PsConn
func (chain *Chain) HandleTrxPsConn(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleTrxPsConn called", chain.groupItem.GroupId)

	//only producer(owner) need handle trx msg from psconn (to build block)
	if !chain.IsProducer() {
		return nil
	}

	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Warningf("trx Version mismatch trx_id <%s>: <%s> vs <%s>", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return fmt.Errorf("trx Version mismatch")
	}

	// decompress
	content := new(bytes.Buffer)
	if err := utils.Decompress(bytes.NewReader(trx.Data), content); err != nil {
		chain_log.Errorf("utils.Decompress failed: %s", err)
		return fmt.Errorf("utils.Decompress failed: %s", err)
	}
	trx.Data = content.Bytes()

	verified, err := rumchaindata.VerifyTrx(trx)
	if err != nil {
		chain_log.Warningf("<%s> verify Trx failed with err <%s>", chain.groupItem.GroupId, err.Error())
		return fmt.Errorf("verify trx failed")
	}

	if !verified {
		chain_log.Warningf("<%s> invalid Trx, signature verify failed, sender <%s>", chain.groupItem.GroupId, trx.SenderPubkey)
		return fmt.Errorf("invalid trx, signature verify failed")
	}

	switch trx.Type {
	case
		quorumpb.TrxType_POST,
		quorumpb.TrxType_UPD_GRP_SYNCER,
		quorumpb.TrxType_CHAIN_CONFIG,
		quorumpb.TrxType_APP_CONFIG,
		quorumpb.TrxType_FORK:
		chain.producerAddTrx(trx)
	default:
		chain_log.Warningf("<%s> unsupported msg type", chain.groupItem.GroupId)
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func (chain *Chain) producerAddTrx(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> producerAddTrx called", chain.groupItem.GroupId)

	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		chain_log.Warningf("<%s> producerAddTrx failed, consensus or producer is nil", chain.groupItem.GroupId)
		return nil
	}

	if !chain.IsProducer() {
		chain_log.Warningf("<%s> producerAddTrx failed, not producer", chain.groupItem.GroupId)
		return nil
	}

	chain.Consensus.Producer().AddTrxToTxBuffer(trx)
	return nil
}

// handle block msg from PSconn
func (chain *Chain) HandleBlockPsConn(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> HandleBlockPsConn called", chain.groupItem.GroupId)

	//check if block is from a valid group producer
	if !chain.IsProducerByPubkey(block.ProducerPubkey) {
		chain_log.Warningf("<%s> received blockid <%d> from unknown producer <%s>, reject it", chain.groupItem.GroupId, block.BlockId, block.ProducerPubkey)
		return nil
	}

	err := chain.Consensus.User().AddBlock(block)
	if err != nil {
		chain_log.Debugf("<%s> FULLNODE add block error <%s>", chain.groupItem.GroupId, err.Error())
		if err.Error() == "PARENT_NOT_EXIST" {
			chain_log.Infof("<%s> block parent not exist, blockId <%s>, currBlockId <%d>, wait syncing",
				chain.groupItem.GroupId, block.BlockId, chain.GetCurrBlockId())
		}
	}

	return nil
}

func (chain *Chain) HandleBftMsgPsConn(msg *quorumpb.BftMsg) error {
	//chain_log.Debugf("<%s> HandleHBPTPsConn called", chain.groupItem.GroupId)
	return chain.Consensus.Producer().HandleBftMsg(msg)
}

// handler SyncMsg from rex
func (chain *Chain) HandleSyncMsgRex(syncMsg *quorumpb.SyncMsg, s network.Stream) error {
	chain_log.Debugf("<%s> HandleTrxRex called", chain.groupItem.GroupId)

	// decompress
	content := new(bytes.Buffer)
	if err := utils.Decompress(bytes.NewReader(syncMsg.Data), content); err != nil {
		e := fmt.Errorf("utils.Decompress failed: %s", err)
		chain_log.Error(e)
		return e
	}

	syncMsg.Data = content.Bytes()

	switch syncMsg.Type {
	case quorumpb.SyncMsgType_REQ_BLOCK:
		chain.handleReqBlockRex(syncMsg, s)
	case quorumpb.SyncMsgType_REQ_BLOCK_RESP:
		chain.handleReqBlockRespRex(syncMsg)
	}
	return nil
}

func (chain *Chain) HandleBroadcastMsgPsConn(brd *quorumpb.BroadcastMsg) error {
	chain_log.Debugf("<%s> HandleGroupBroadcastPsConn called", chain.groupItem.GroupId)
	return nil
}

func (chain *Chain) handleReqBlockRex(syncMsg *quorumpb.SyncMsg, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlocks called", chain.groupItem.GroupId)
	//unmarshall req
	req := &quorumpb.ReqBlock{}
	err := proto.Unmarshal(syncMsg.Data, req)
	if err != nil {
		chain_log.Warningf("<%s> handleReqBlocksRex error <%s>", chain.groupItem.GroupId, err.Error())
		return err
	}

	//do nothing is req is from myself
	if req.ReqPubkey == chain.groupItem.UserSignPubkey {
		return nil
	}

	//check if req sender is allow to sync
	if !(chain.groupItem.SyncType == quorumpb.GroupSyncType_PUBLIC) && !chain.IsGroupSyncer(req.ReqPubkey) {
		chain_log.Warningf("<%s> handleReqBlocksRex error <%s>", chain.groupItem.GroupId, "not a syncer")
		return nil
	}

	//verify req
	verified, err := rumchaindata.VerifyReqBlock(req)
	if err != nil {
		chain_log.Warningf("<%s> verify ReqBlock failed with err <%s>", chain.groupItem.GroupId, err.Error())
		return err
	}

	if !verified {
		chain_log.Warningf("<%s> Invalid ReqBlock, signature verify failed, sender <%s>", chain.groupItem.GroupId, req.ReqPubkey)
		return errors.New("invalid ReqBlock")
	}

	//get resp
	blocks, result, err := chain.chaindata.GetReqBlocks(req)
	if err != nil {
		return err
	}

	chain_log.Debugf("<%s> send REQ_BLOCKS_RESP", chain.groupItem.GroupId)
	chain_log.Debugf("-- requester <%s>, from Block <%d>, request <%d> blocks", req.ReqPubkey, req.FromBlock, req.BlksRequested)
	chain_log.Debugf("-- send fromBlock <%d>, total <%d> blocks, status <%s>", req.FromBlock, len(blocks), result.String())

	//resp, err = chain.trxFactory.GetReqBlocksRespTrx("", chain.groupItem.GroupId, requester, fromBlock, blkReqs, blocks, result)
	resp, err := rumchaindata.GetReqBlocksRespMsg("", req, chain.groupItem.UserSignPubkey, blocks, result)

	if err != nil {
		return err
	}

	if cmgr, err := conn.GetConn().GetConnMgr(chain.groupItem.GroupId); err != nil {
		return err
	} else {
		return cmgr.SendSyncRespMsgRex(resp, s)
	}
}

func (chain *Chain) handleReqBlockRespRex(syncMsg *quorumpb.SyncMsg) error {
	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupItem.GroupId)

	resp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(syncMsg.Data, resp); err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.groupItem.GroupId, err.Error())
		return err
	}

	//if not asked by me, ignore it
	if resp.RequesterPubkey != chain.groupItem.UserSignPubkey {
		//chain_log.Debugf("<%s> HandleReqBlockResp error <%s>", chain.Group.GroupId, rumerrors.ErrSenderMismatch.Error())
		return nil
	}

	//verify resp
	verified, err := rumchaindata.VerifyReqBlockResp(resp)
	if err != nil {
		chain_log.Warningf("<%s> verify ReqBlockResp failed with err <%s>", chain.groupItem.GroupId, err.Error())
		return err
	}

	if !verified {
		chain_log.Warningf("<%s> Invalid ReqBlockResp, signature verify failed, sender <%s>", chain.groupItem.GroupId, resp.ProviderPubkey)
		return errors.New("invalid ReqBlockResp")
	}

	result := &SyncResult{
		TaskId: resp.FromBlock,
		Data:   resp,
	}

	chain.rexSyncer.AddResult(result)
	return nil
}

func (chain *Chain) ApplyBlocks(blocks []*quorumpb.Block) error {
	chain_log.Warningf("<%s> TODO: add a lock in ApplyBlocks()", chain.groupItem.GroupId)
	//RumLite Node Add synced Block
	for _, block := range blocks {
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Warningf("<%s> ApplyBlocks error <%s>", chain.groupItem.GroupId, err.Error())
			return err
		}
	}
	return nil
}

func (chain *Chain) updSyncerList() {
	chain_log.Debugf("<%s> updSyncerList called", chain.groupItem.GroupId)
	//create and load Group user pool
	chain.syncerPool = make(map[string]*quorumpb.Syncer)
	syncers, err := nodectx.GetNodeCtx().GetChainStorage().GetSyncers(chain.groupItem.GroupId, chain.nodename)
	if err != nil {
		chain_log.Debugf("Get syncers with err <%s>", err.Error())
		return
	}

	for _, syncer := range syncers {
		chain.syncerPool[syncer.SyncerPubkey] = syncer
		chain_log.Infof("<%s> load syncer <%s>", chain.groupItem.GroupId, syncer.SyncerPubkey)
	}
}

func (chain *Chain) updateProducerPool() {
	chain_log.Debugf("<%s> UpdProducerList called", chain.groupItem.GroupId)
	chain.producerPool = make(map[string]*quorumpb.ProducerItem)
	producers, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(chain.groupItem.GroupId, chain.nodename)

	if err != nil {
		chain_log.Debugf("Get producer failed with err <%s>", err.Error())
	}

	for _, item := range producers {
		chain.producerPool[item.ProducerPubkey] = item
		isOwner := ""
		if item.ProducerPubkey == chain.groupItem.OwnerPubKey {
			isOwner = "(owner)"
		}
		chain_log.Debugf("<%s> load producer <%s%s>", chain.groupItem.GroupId, item.ProducerPubkey, isOwner)
	}
}

func (chain *Chain) CreateConsensus() error {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupItem.GroupId)

	var user def.User
	var producer def.Producer

	chain_log.Infof("<%s> Create and initial molasses producer", chain.groupItem.GroupId)
	producer = &consensus.MolassesProducer{}
	producer.NewProducer(chain.ChainCtx, chain.groupItem, chain.nodename, chain)

	chain_log.Infof("<%s> Create and initial molasses user", chain.groupItem.GroupId)
	user = &consensus.MolassesUser{}
	user.NewUser(chain.groupItem, chain.nodename, chain)

	chain.Consensus = consensus.NewMolasses(producer, user)
	chain.Consensus.StartProposeTrx()

	return nil
}

func (chain *Chain) IsProducer() bool {
	_, ok := chain.producerPool[chain.groupItem.UserSignPubkey]
	return ok
}

func (chain *Chain) IsProducerByPubkey(pubkey string) bool {
	_, ok := chain.producerPool[pubkey]
	return ok
}

func (chain *Chain) IsOwner() bool {
	return chain.groupItem.OwnerPubKey == chain.groupItem.UserSignPubkey
}

func (chain *Chain) IsOwnerByPubkey(pubkey string) bool {
	return chain.groupItem.OwnerPubKey == pubkey
}

func (chain *Chain) IsGroupSyncer(pubkey string) bool {
	_, ok := chain.syncerPool[pubkey]
	return ok
}

func (chain *Chain) GetRexSyncerStatus() string {
	status := chain.rexSyncer.GetSyncerStatus()
	statusStr := ""

	//cast status to string
	switch status {
	case IDLE:
		statusStr = "IDLE"
	case RUNNING:
		statusStr = "SYNCING"
	case CLOSED:
		statusStr = "CLOSED"
	default:

	}
	return statusStr
}

func (chain *Chain) GetLastRexSyncResult() (*chaindef.RexSyncResult, error) {
	return chain.rexSyncer.GetLastRexSyncResult()
}

func (chain *Chain) ApplyTrxsRumLiteNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsFullNode called", chain.groupItem.GroupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, nodename)
		if err != nil {
			chain_log.Warningf("<%s> check trx <%s> exist failed with error <%s>", chain.groupItem.GroupId, trx.TrxId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> already applied, skip", chain.groupItem.GroupId, trx.TrxId)
			continue
		}

		//verify trx
		isTrxValid, err := rumchaindata.VerifyTrx(trx)
		if err != nil {
			chain_log.Warningf("<%s> verify trx <%s> failed with error <%s>", chain.groupItem.GroupId, trx.TrxId, err.Error())
			continue
		}

		if !isTrxValid {
			chain_log.Warningf("<%s> trx <%s> is not valid", chain.groupItem.GroupId, trx.TrxId)
			continue
		}

		//check if these type of TRX is send by owner
		if trx.Type == quorumpb.TrxType_CHAIN_CONFIG ||
			trx.Type == quorumpb.TrxType_FORK ||
			trx.Type == quorumpb.TrxType_UPD_GRP_SYNCER {
			if !chain.IsOwnerByPubkey(trx.SenderPubkey) {
				chain_log.Warningf("<%s> trx <%s> with type <%s> is not send by owner, skip", chain.groupItem.GroupId, trx.TrxId, trx.Type.String())
				continue
			}
		}

		//decode data
		ciperKey, err := hex.DecodeString(chain.groupItem.CipherKey)
		if err != nil {
			return err
		}

		decodedData, err := localcrypto.AesDecode(trx.Data, ciperKey)
		if err != nil {
			return err
		}

		chain_log.Debugf("<%s> try apply trx <%s>", chain.groupItem.GroupId, trx.TrxId)
		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Debugf("<%s> apply POST trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().AddPost(trx, decodedData, nodename)
		case quorumpb.TrxType_UPD_GRP_SYNCER:
			chain_log.Debugf("<%s> apply UPD_GRP_SYNCER trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateGroupSyncer(trx.TrxId, decodedData, nodename)
			chain.updSyncerList()
		case quorumpb.TrxType_APP_CONFIG:
			chain_log.Debugf("<%s> apply APP_CONFIG trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAppConfig(decodedData, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfig(decodedData, nodename)
		//case quorumpb.TrxType_CONSENSUS:
		//	chain_log.Debugf("<%s> apply CONSENSUS trx", chain.groupItem.GroupId)
		//	chain.applyConseususTrx(trx, decodedData, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.groupItem.GroupId, trx.Type.String())
		}

		//save original trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}
	return nil
}

func (chain *Chain) VerifySign(hash, signature []byte, pubkey string) (bool, error) {
	//check signature
	bytespubkey, err := base64.RawURLEncoding.DecodeString(pubkey)
	if err != nil {
		return false, err
	}
	ethpbukey, err := ethcrypto.DecompressPubkey(bytespubkey)
	if err == nil {
		ks := localcrypto.GetKeystore()
		r := ks.EthVerifySign(hash, signature, ethpbukey)
		if !r {
			return false, fmt.Errorf("verify signature failed")
		}
	} else {
		return false, err
	}

	return true, nil
}

func (chain *Chain) StartSync() error {
	chain_log.Debugf("<%s> StartSync called", chain.groupItem.GroupId)
	chain.rexSyncer.Start()
	return nil
}

func (chain *Chain) StopSync() {
	chain_log.Debugf("<%s> StopSync called", chain.groupItem.GroupId)
	if chain.rexSyncer != nil {
		chain.rexSyncer.Stop()
	}
}

func (chain *Chain) GetBlockFromDSCache(groupId string, blockId uint64, prefix ...string) (*quorumpb.Block, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetBlockFromDSCache(groupId, blockId, chain.nodename)
}

//local sync
//TODO
//func (chain *Chain) SyncLocalBlock() error {
//	startFrom := chain.Group.Item.HighestBlockId
//	for {
//		subblocks, err := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(chain.Group.Item.HighestBlockId, chain.nodename)
//		if err != nil {
//			chain_log.Debugf("<%s> GetSubBlock failed <%s>", chain.GroupId, err.Error())
//			return err
//		}
//		if len(subblocks) > 0 {
//			for _, block := range subblocks {
//				err := chain.AddLocalBlock(block)
//				if err != nil {
//					chain_log.Debugf("<%s> AddLocalBlock failed <%s>", chain.GroupId, err.Error())
//					break // for range subblocks
//				}
//			}
//		} else {
//			chain_log.Debugf("<%s> No more local blocks", chain.GroupId)
//			return nil
//		}
//		topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(chain.Group.Item.HighestBlockId, false, chain.nodename)
//		if err != nil {
//			chain_log.Debugf("<%s> Get Top Block failed <%s>", chain.GroupId, err.Error())
//			return err
//		} else {
//			if topBlock.BlockId == startFrom {
//				return nil
//			} else {
//				startFrom = topBlock.BlockId
//			}
//		}
//	}
//
//}

//TODO
//func (chain *Chain) AddLocalBlock(block *quorumpb.Block) error {
//	chain_log.Debugf("<%s> AddLocalBlock called", chain.GroupId)
//	signpkey, err := localcrypto.Libp2pPubkeyToEthBase64(chain.Group.Item.UserSignPubkey)
//	if err != nil && signpkey == "" {
//		chain_log.Warnf("<%s> Pubkey err <%s>", chain.GroupId, err)
//	}
//
//	_, producer := chain.ProducerPool[signpkey]
//
//	if producer {
//		chain_log.Debugf("<%s> PRODUCER ADD LOCAL BLOCK <%d>", chain.GroupId, block.Epoch)
//		err := chain.AddBlock(block)
//		if err != nil {
//			chain_log.Infof(err.Error())
//		}
//	} else {
//		chain_log.Debugf("<%s> USER ADD LOCAL BLOCK <%d>", chain.GroupId, block.Epoch)
//		err := chain.Consensus.User().AddBlock(block)
//		if err != nil {
//			chain_log.Infof(err.Error())
//		}
//	}
//	return nil
//}

package chain

import (
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

type Chain struct {
	groupItem    *quorumpb.GroupItem
	nodename     string
	producerPool map[string]*quorumpb.ProducerItem
	userPool     map[string]*quorumpb.UserItem
	trxFactory   *rumchaindata.TrxFactory
	rexSyncer    *RexSyncer
	chaindata    *ChainData
	Consensus    def.Consensus
	CurrBlock    uint64
	CurrEpoch    uint64
	LatestUpdate int64
}

func (chain *Chain) NewChain(item *quorumpb.GroupItem, nodename string, loadChainInfo bool) error {
	chain_log.Debugf("<%s> NewChain called", item.GroupId)

	chain.groupItem = item
	chain.nodename = nodename

	//initial TrxFactory
	chain.trxFactory = &rumchaindata.TrxFactory{}
	chain.trxFactory.Init(nodectx.GetNodeCtx().Version, chain.groupItem, chain.nodename, chain)

	//initial Syncer
	chain.rexSyncer = NewRexSyncer(chain.groupItem.GroupId, chain.nodename, chain, chain)

	//initial chaindata manager
	chain.chaindata = &ChainData{
		nodename:       chain.nodename,
		groupId:        chain.groupItem.GroupId,
		groupCipherKey: chain.groupItem.CipherKey,
		userSignPubkey: chain.groupItem.UserSignPubkey,
		dbmgr:          nodectx.GetDbMgr()}

	if loadChainInfo {
		currBlockId, currEpoch, lastUpdate, err := nodectx.GetNodeCtx().GetChainStorage().GetChainInfo(chain.groupItem.GroupId, chain.nodename)
		if err != nil {
			return err
		}
		chain_log.Debugf("<%s> Set epoch <%d>, set lastUpdate <%d>", chain.groupItem.GroupId, currEpoch, lastUpdate)
		chain.SetCurrEpoch(currEpoch)
		chain.SetLastUpdate(lastUpdate)
		chain.SetCurrBlockId(currBlockId)
	} else {
		currEpoch := uint64(0)
		currBlock := uint64(0)
		lastUpdate := time.Now().UnixNano()
		chain_log.Debugf("<%s> Set epoch <%d>, set lastUpdate <%d>", chain.groupItem.GroupId, currEpoch, lastUpdate)
		chain.SetCurrEpoch(currEpoch)
		chain.SetCurrBlockId(currBlock)
		chain.SetLastUpdate(lastUpdate)
		chain.SaveChainInfoToDb()
	}

	chain_log.Debugf("<%s> NewChain done", chain.groupItem.GroupId)

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
	chain_log.Debugf("<%s> SaveChainInfoToDb called", chain.groupItem.GroupId)
	chain_log.Debugf("<%s> Current Epoch <%d>, lastUpdate <%d>", chain.groupItem.GroupId, chain.GetCurrEpoch(), chain.GetLastUpdate())
	return nodectx.GetNodeCtx().GetChainStorage().SaveChainInfo(chain.GetCurrBlockId(), chain.GetCurrEpoch(), chain.GetLastUpdate(), chain.groupItem.GroupId, chain.nodename)
}

func (chain *Chain) GetTrxFactory() chaindef.TrxFactoryIface {
	chain_log.Debugf("<%s> GetTrxFactory called", chain.groupItem.GroupId)
	return chain.trxFactory
}

func (chain *Chain) GetPubqueueIface() chaindef.PublishQueueIface {
	chain_log.Debugf("<%s> GetPubqueueIface called", chain.groupItem.GroupId)
	return GetPubQueueWatcher()
}

func (chain *Chain) ReqPSync() (string, error) {
	chain_log.Debugf("<%s> ReqPSync called, TBD", chain.groupItem.GroupId)
	//return chain.syncerrunner.GetPSync()
	return "", nil
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
			err = chain.HandleBlockPsConn(blk)
		}
	} else if pkg.Type == quorumpb.PackageType_TRX {
		trx := &quorumpb.Trx{}
		err = proto.Unmarshal(pkg.Data, trx)
		if err != nil {
			chain_log.Warningf(err.Error())
		} else {
			err = chain.HandleTrxPsConn(trx)
		}
	} else if pkg.Type == quorumpb.PackageType_HBB {
		hb := &quorumpb.HBMsgv1{}
		err = proto.Unmarshal(pkg.Data, hb)
		if err != nil {
			chain_log.Warningf(err.Error())
		} else {
			err = chain.HandleHBPsConn(hb)
		}
	} else if pkg.Type == quorumpb.PackageType_PSYNC {
		psync := &quorumpb.PSyncMsg{}
		err = proto.Unmarshal(pkg.Data, psync)
		if err != nil {
			chain_log.Warnf(err.Error())
		} else {
			err = chain.HandlePSyncPsConn(psync)
		}
	}

	return err
}

// Handle Trx from PsConn
func (chain *Chain) HandleTrxPsConn(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleTrxPsConn called", chain.groupItem.GroupId)

	//only producer(owner) need handle trx msg from psconn (to build trxs into block)
	if !chain.isProducer() {
		//chain_log.Infof("non producer(owner) ignore incoming trx from psconn")
		return nil
	}

	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Warningf("trx Version mismatch trx_id <%s>: <%s> vs <%s>", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return fmt.Errorf("trx Version mismatch")
	}

	verified, err := rumchaindata.VerifyTrx(trx)
	if err != nil {
		chain_log.Warningf("<%s> verify Trx failed with err <%s>", chain.groupItem.GroupId, err.Error())
		return fmt.Errorf("verify Trx failed")
	}

	if !verified {
		chain_log.Warningf("<%s> invalid Trx, signature verify failed, sender <%s>", chain.groupItem.GroupId, trx.SenderPubkey)
		return fmt.Errorf("invalid Trx")
	}

	switch trx.Type {
	case
		quorumpb.TrxType_POST,
		quorumpb.TrxType_ANNOUNCE,
		quorumpb.TrxType_PRODUCER,
		quorumpb.TrxType_USER,
		quorumpb.TrxType_APP_CONFIG,
		quorumpb.TrxType_CHAIN_CONFIG:
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
		return nil
	}

	chain.Consensus.Producer().AddTrx(trx)
	return nil
}

// handle block msg from PSconn
func (chain *Chain) HandleBlockPsConn(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> HandleBlockPsConn called", chain.groupItem.GroupId)

	// all approved producers(owner) should ignore block from psconn (they gonna build block by themselves)
	// when sync, for all node blocks will come from rex channel
	if chain.isProducer() {
		//chain_log.Infof("producer(owner) ignore incoming block from psconn")
		return nil
	}

	//DONT CHECK THIS
	//check if block is from approved producer
	/*
		if !chain.isProducerByPubkey(block.ProducerPubkey) {
			chain_log.Warningf("<%s> received block <%d> from unapproved producer <%s>, reject it", chain.groupItem.GroupId, block.Epoch, block.ProducerPubkey)
			return nil
		}
	*/

	//for all node run as PRODUCER_NODE but not approved by owner (yet)
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		chain_log.Debugf("<%s> producer node add block", chain.groupItem.GroupId)
		err := chain.Consensus.Producer().AddBlock(block)
		if err != nil {
			chain_log.Warningf("<%s> announced producer add block error <%s>", chain.groupItem.GroupId, err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				chain_log.Debugf("<%s> announced producer add block, parent not exist, blockId <%d>, currBlockId <%d>",
					chain.groupItem.GroupId, block.BlockId, chain.GetCurrBlockId())
			}
		}
		return err
	}

	//for all node run as FULLNODE (except owner)
	err := chain.Consensus.User().AddBlock(block)
	if err != nil {
		chain_log.Debugf("<%s> FULLNODE add block error <%s>", chain.groupItem.GroupId, err.Error())
		if err.Error() == "PARENT_NOT_EXIST" {
			chain_log.Infof("<%s> block parent not exist, blockId <%s>, currBlockId <%d>",
				chain.groupItem.GroupId, block.BlockId, chain.GetCurrBlockId())
		}
	}

	return nil
}

// handle HBB msg from PsConn
func (chain *Chain) HandleHBPsConn(hb *quorumpb.HBMsgv1) error {
	//chain_log.Debugf("<%s> HandleHBPsConn called", chain.groupItem.GroupId)

	//only producers(owner) need to handle HBB message
	if !chain.isProducer() {
		return nil
	}

	if chain.Consensus.Producer() == nil {
		chain_log.Warningf("<%s> Consensus Producer is null", chain.groupItem.GroupId)
		return nil
	}
	return chain.Consensus.Producer().HandleHBMsg(hb)
}

// handle psync consensus req from PsConn
func (chain *Chain) HandlePSyncPsConn(psync *quorumpb.PSyncMsg) error {
	chain_log.Debugf("<%s> HandlePSyncConsesusReqPsConn called", chain.groupItem.GroupId)

	//only producers(owner) need to handle Consensus msg
	if !chain.isProducer() {
		return nil
	}

	if chain.Consensus.PSync() == nil {
		chain_log.Warningf("<%s> Consensus PSync is null", chain.groupItem.GroupId)
		return nil
	}

	if psync.MsgType == quorumpb.PSyncMsgType_PSYNC_REQ {
		//handle psyncReqmsg
		req := &quorumpb.PSyncReq{}
		err := proto.Unmarshal(psync.Payload, req)
		if err != nil {
			return err
		}

		if !chain.isProducerByPubkey(req.SenderPubkey) {
			chain_log.Warningf("<%s> PSyncReq from non producer node <%s>, ignore", chain.groupItem.GroupId, req.SenderPubkey)
			return nil
		}

		//verify signature
		reqWithoutSign := &quorumpb.PSyncReq{
			GroupId:      req.GroupId,
			SessionId:    req.SessionId,
			SenderPubkey: req.SenderPubkey,
			MyEpoch:      req.MyEpoch,
		}

		db, _ := proto.Marshal(reqWithoutSign)
		dhash := localcrypto.Hash(db)
		verifySign, err := chain.VerifySign(dhash, req.SenderSign, req.SenderPubkey)
		if err != nil {
			return err
		}
		if !verifySign {
			return fmt.Errorf("verify signature failed")
		}

		//let psync handle the req
		return chain.Consensus.PSync().AddPSyncReq(req)
	} else if psync.MsgType == quorumpb.PSyncMsgType_PSYNC_RESP {
		resp := &quorumpb.PSyncResp{}
		err := proto.Unmarshal(psync.Payload, resp)
		if err != nil {
			return err
		}

		if !chain.isProducerByPubkey(resp.SenderPubkey) {
			chain_log.Warningf("<%s> PSyncResp from non producer node <%s>, ignore", chain.groupItem.GroupId, resp.SenderPubkey)
			return nil
		}

		//check if the resp is from myself
		if len(chain.producerPool) != 1 && chain.groupItem.UserSignPubkey == resp.SenderPubkey {
			chain_log.Debugf("multiple producer exist, session <%s> consensusResp from myself, ignore", resp.SessionId)
			return nil
		}

		//check if psync result with same session_id exist
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsPSyncSessionExist(chain.groupItem.GroupId, resp.SessionId)
		if err != nil {
			return err
		}

		if isExist {
			chain_log.Debugf("Session <%s> is handled, ignore", resp.SessionId)
			return nil
		}

		//verify response
		respWithoutSign := &quorumpb.PSyncResp{
			GroupId:           resp.GroupId,
			SessionId:         resp.SessionId,
			SenderPubkey:      resp.SenderPubkey,
			MyCurEpoch:        resp.MyCurEpoch,
			MyCurProducerList: resp.MyCurProducerList,
			ProducerProof:     resp.ProducerProof,
		}

		//get hash
		db, _ := proto.Marshal(respWithoutSign)
		if err != nil {
			return err
		}
		dhash := localcrypto.Hash(db)

		//verify signature
		verifySign, err := chain.VerifySign(dhash, resp.SenderSign, resp.SenderPubkey)
		if err != nil {
			return err
		}
		if !verifySign {
			return fmt.Errorf("verify signature failed")
		}

		ok, err := chain.verifyProducer(resp.SenderPubkey, resp)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("invalid consensusResp from producer <%s>", resp.SenderPubkey)
		}

		return chain.handlePSyncResp(resp)
	}

	return fmt.Errorf("unknown msgType <%s>", psync.MsgType.String())

}

func (chain *Chain) handlePSyncResp(resp *quorumpb.PSyncResp) error {
	chain_log.Debugf("<%s> handlePSyncResp called, SessionId <%s>", chain.groupItem.GroupId, resp.SessionId)

	/*

			//check if the resp is what gsync expected
			taskId, taskType, _, err := chain.syncerrunner.GetCurrentSyncTask()
			if err == rumerrors.ErrNoTaskWait || taskType != PSync || taskId != resp.SessionId {
				//not the expected consensus resp
				return rumerrors.ErrConsusMismatch
			}

				savedResp, err := nodectx.GetNodeCtx().GetChainStorage().GetCurrentPSyncSession(chain.groupItem.GroupId)
				if err != nil {
					return err
				}


				//just in case
				if len(savedResp) != 1 {
					chain_log.Warningf("<%s> get <%d> saved psync resp msg (should be 1), something goes wrong", chain.groupItem.GroupId, len(savedResp))
					return fmt.Errorf("psync resp msg mismatch, something goes wrong")
				}

				respItem := savedResp[0]
				if respItem.CurChainEpoch > resp.CurChainEpoch {
					chain_log.Debugf("resp from old epoch, do nothing, ignore")
					return fmt.Errorf("resp from old epoch, ignore")
				}



				//TBD check and update producer according to psync resp
				/*
					trx, _, err := nodectx.GetNodeCtx().GetChainStorage().GetTrx(chain.GroupId, resp.ProducerProof.TrxId, sdef.Chain, chain.nodename)
					if err != nil && trx != nil {
						chain_log.Debugf("No need to upgrade producer list")
					} else {
						//TBD update producers list and regerate all consensus
						// user
						// producer
						// psync
					}


		//save ConsensusResp
		//nodectx.GetNodeCtx().GetChainStorage().UpdPSyncResp(chain.groupItem.GroupId, sessionId, resp)

		if resp.MyCurEpoch == chain.GetCurrEpoch() {
			chain_log.Debugf("node local epoch == current chain epoch, No need to sync")
			chain.syncerrunner.UpdatePSyncResult(resp.SessionId, uint(SyncDone))
		} else {
			chain.syncerrunner.UpdatePSyncResult(resp.SessionId, uint(ContinueGetEpoch))
		}

	*/

	return nil
}

func (chain *Chain) verifyProducer(senderPubkey string, resp *quorumpb.PSyncResp) (bool, error) {
	chain_log.Debugf("<%s> verifyProducer called", chain.groupItem.GroupId)

	//consensusResp from owner, trust it anyway
	if senderPubkey == chain.groupItem.OwnerPubKey {
		chain_log.Debugf("PSyncResp sender <%s> is owner", senderPubkey)
		return true, nil
	}

	//conosensusResp form other producer, in this resp,
	//no other producers approved (owner works as the only Group producer)
	//no related trx to be verified
	if len(resp.MyCurProducerList.Producers) == 1 && resp.MyCurProducerList.Producers[0] == chain.groupItem.OwnerPubKey {
		return true, nil
	}

	//verify related PRODUCER trx as a proof
	trxOK, err := rumchaindata.VerifyTrx(resp.ProducerProof)
	if err != nil {
		return false, err
	}

	if !trxOK {
		chain_log.Debugf("invalid trx")
		return false, err
	}

	//decode trx data by using ciperKey
	ciperKey, err := hex.DecodeString(chain.groupItem.CipherKey)
	if err != nil {
		return false, err
	}

	encryptdData, err := localcrypto.AesDecode(resp.ProducerProof.Data, ciperKey)
	if err != nil {
		return false, err
	}

	bftProducerBundleItem := &quorumpb.BFTProducerBundleItem{}
	err = proto.Unmarshal(encryptdData, bftProducerBundleItem)
	if err != nil {
		return false, err
	}

	//sender(producer) pubkey should in the update producer trx list
	for _, producer := range bftProducerBundleItem.Producers {
		if producer.ProducerPubkey == senderPubkey {
			chain_log.Debugf("consensus sender <%s> is valid producer", senderPubkey)
			return true, nil
		}
	}

	//no, not a producer
	return false, nil
}

func (chain *Chain) HandlePSyncRex(c *quorumpb.PSyncMsg) error {
	return nil
}

// handler trx from rex (for sync only)
func (chain *Chain) HandleTrxRex(trx *quorumpb.Trx, s network.Stream) error {
	chain_log.Debugf("<%s> HandleTrxRex called", chain.groupItem.GroupId)
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Warningf("HandleTrxRex called, Trx Version mismatch, trxid <%s>: <%s> vs <%s>", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return fmt.Errorf("trx Version mismatch")
	}

	//TBD should check if requester from block list

	verified, err := rumchaindata.VerifyTrx(trx)
	if err != nil {
		chain_log.Warningf("<%s> verify Trx failed with err <%s>", chain.groupItem.GroupId, err.Error())
		return fmt.Errorf("verify Trx failed")
	}

	if !verified {
		chain_log.Warnf("<%s> Invalid Trx, signature verify failed, sender <%s>", chain.groupItem.GroupId, trx.SenderPubkey)
		return fmt.Errorf("invalid Trx")
	}

	if trx.SenderPubkey == chain.groupItem.UserSignPubkey {
		//ignore msg from myself
		return nil
	}

	//Rex Channel only support the following trx type
	switch trx.Type {
	case quorumpb.TrxType_REQ_BLOCK:
		chain.handleReqBlocks(trx, s)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		chain.handleReqBlockResp(trx)
	default:
		//do nothing
	}

	return nil
}

// ununsed
func (chain *Chain) HandleBlockRex(block *quorumpb.Block, s network.Stream) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.groupItem.GroupId)
	return nil
}

// unused
func (chain *Chain) HandleHBRex(hb *quorumpb.HBMsgv1) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.groupItem.GroupId)
	return nil
}

func (chain *Chain) handleReqBlocks(trx *quorumpb.Trx, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlocks called", chain.groupItem.GroupId)
	requester, fromBlock, blkReqs, blocks, result, err := chain.chaindata.GetReqBlocks(trx)
	if err != nil {
		return err
	}

	chain_log.Debugf("<%s> send REQ_BLOCKS_RESP", chain.groupItem.GroupId)
	chain_log.Debugf("-- requester <%s>, from Block <%d>, request <%d> blocks", requester, fromBlock, blkReqs)
	chain_log.Debugf("-- send fromBlock <%d>, total <%d> blocks, status <%s>", fromBlock, len(blocks), result.String())

	trx, err = chain.trxFactory.GetReqBlocksRespTrx("", chain.groupItem.GroupId, requester, fromBlock, blkReqs, blocks, result)
	if err != nil {
		return err
	}

	if cmgr, err := conn.GetConn().GetConnMgr(chain.groupItem.GroupId); err != nil {
		return err
	} else {
		return cmgr.SendRespTrxRex(trx, s)
	}
}

func (chain *Chain) handleReqBlockResp(trx *quorumpb.Trx) {
	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupItem.GroupId)

	//decode resp
	var err error
	ciperKey, err := hex.DecodeString(chain.groupItem.CipherKey)
	if err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.groupItem.GroupId, err.Error())
		return
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.groupItem.GroupId, err.Error())
		return
	}

	reqBlockResp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(decryptData, reqBlockResp); err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.groupItem.GroupId, err.Error())
		return
	}

	//if not asked by me, ignore it
	if reqBlockResp.RequesterPubkey != chain.groupItem.UserSignPubkey {
		//chain_log.Debugf("<%s> HandleReqBlockResp error <%s>", chain.Group.GroupId, rumerrors.ErrSenderMismatch.Error())
		return
	}

	//check trx sender
	if trx.SenderPubkey != reqBlockResp.ProviderPubkey {
		chain_log.Debugf("<%s> HandleReqBlockResp - Trx Sender/blocks providers mismatch <%s>", chain.groupItem.GroupId)
		return
	}

	result := &SyncResult{
		TaskId: reqBlockResp.FromBlock,
		Data:   reqBlockResp,
	}

	chain.rexSyncer.AddResult(result)
}

func (chain *Chain) ApplyBlocks(blocks []*quorumpb.Block) error {
	//PRODUCER_NODE add SYNC
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		for _, block := range blocks {
			err := chain.Consensus.Producer().AddBlock(block)
			if err != nil {
				chain_log.Warningf("<%s> ApplyBlocks error <%s>", chain.groupItem.GroupId, err.Error())
				return err
			}
		}

		return nil
	}

	//FULLNODE (include owner) Add synced Block
	for _, block := range blocks {
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Warningf("<%s> ApplyBlocks error <%s>", chain.groupItem.GroupId, err.Error())
			return err
		}
	}

	return nil
}

func (chain *Chain) UpdConnMgrProducer() {
	chain_log.Debugf("<%s> UpdConnMgrProducer called", chain.groupItem.GroupId)
	connMgr, _ := conn.GetConn().GetConnMgr(chain.groupItem.GroupId)

	var producerspubkey []string
	for key := range chain.producerPool {
		producerspubkey = append(producerspubkey, key)
	}

	connMgr.UpdProducers(producerspubkey)
}

func (chain *Chain) updProducerList() {
	chain_log.Debugf("<%s> UpdProducerList called", chain.groupItem.GroupId)
	//create and load Group producer pool
	chain.producerPool = make(map[string]*quorumpb.ProducerItem)
	producers, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(chain.groupItem.GroupId, chain.nodename)

	if err != nil {
		chain_log.Infof("Get producer failed with err <%s>", err.Error())
	}

	for _, item := range producers {
		base64ethpkey, err := localcrypto.Libp2pPubkeyToEthBase64(item.ProducerPubkey)
		if err == nil {
			chain.producerPool[base64ethpkey] = item
		} else {
			chain.producerPool[item.ProducerPubkey] = item
		}
		ownerPrefix := "(producer)"
		if item.ProducerPubkey == chain.groupItem.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> load producer <%s%s>", chain.groupItem.GroupId, item.ProducerPubkey, ownerPrefix)
	}
}

func (chain *Chain) updAnnouncedProducerStatus() {
	chain_log.Debugf("<%s> updAnnouncedProducerStatus called", chain.groupItem.GroupId)

	//update announced producer result
	announcedProducers, _ := nodectx.GetNodeCtx().GetChainStorage().GetAnnounceProducersByGroup(chain.groupItem.GroupId, chain.nodename)
	for _, item := range announcedProducers {
		_, ok := chain.producerPool[item.SignPubkey]
		err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounceResult(quorumpb.AnnounceType_AS_PRODUCER, chain.groupItem.GroupId, item.SignPubkey, ok, chain.nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.groupItem.GroupId, err.Error())
		}
	}
}

func (chain *Chain) updProducerConfig() {
	chain_log.Debugf("<%s> updProducerConfig called", chain.groupItem.GroupId)
	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return
	}

	//recreate producer BFT config
	chain.Consensus.Producer().RecreateBft()
}

func (chain *Chain) updUserList() {
	chain_log.Debugf("<%s> updUserList called", chain.groupItem.GroupId)

	//create and load Group user pool
	chain.userPool = make(map[string]*quorumpb.UserItem)
	users, _ := nodectx.GetNodeCtx().GetChainStorage().GetUsers(chain.groupItem.GroupId, chain.nodename)
	for _, item := range users {
		chain.userPool[item.UserPubkey] = item
		ownerPrefix := "(user)"
		if item.UserPubkey == chain.groupItem.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> Load Users <%s_%s>", chain.groupItem.GroupId, item.UserPubkey, ownerPrefix)
	}

	//update announced User result
	announcedUsers, _ := nodectx.GetNodeCtx().GetChainStorage().GetAnnounceUsersByGroup(chain.groupItem.GroupId, chain.nodename)
	for _, item := range announcedUsers {
		_, ok := chain.userPool[item.SignPubkey]
		err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounceResult(quorumpb.AnnounceType_AS_USER,
			chain.groupItem.GroupId,
			item.SignPubkey,
			ok,
			chain.nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.groupItem.GroupId, err.Error())
		}
	}
}

func (chain *Chain) GetUsesEncryptPubKeys() ([]string, error) {
	keys := []string{}
	ks := nodectx.GetNodeCtx().Keystore
	mypubkey, err := ks.GetEncodedPubkey(chain.groupItem.GroupId, localcrypto.Encrypt)
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

func (chain *Chain) CreateConsensus() error {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupItem.GroupId)

	var user def.User
	var producer def.Producer
	var psync def.PSync

	var shouldCreateUser, shouldCreateProducer, shouldCreatePSyncer bool

	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		shouldCreateProducer = true
		shouldCreateUser = false
		shouldCreatePSyncer = true
	} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
		//check if I am owner of the Group
		if chain.groupItem.UserSignPubkey == chain.groupItem.OwnerPubKey {
			shouldCreateProducer = true
			shouldCreatePSyncer = true
		} else {
			shouldCreateProducer = false
			shouldCreatePSyncer = false
		}
		shouldCreateUser = true
	} else {
		return fmt.Errorf("unknow nodetype")
	}

	if shouldCreateProducer {
		chain_log.Infof("<%s> Create and initial molasses producer", chain.groupItem.GroupId)
		producer = &consensus.MolassesProducer{}
		producer.NewProducer(chain.groupItem, chain.nodename, chain)
		producer.StartPropose()
	}

	if shouldCreateUser {
		chain_log.Infof("<%s> Create and initial molasses user", chain.groupItem.GroupId)
		user = &consensus.MolassesUser{}
		user.NewUser(chain.groupItem, chain.nodename, chain)
	}

	if shouldCreatePSyncer {
		chain_log.Infof("<%s> Create and initial molasses psyncer", chain.groupItem.GroupId)
		psync = &consensus.MolassesPSync{}
		psync.NewPSync(chain.groupItem, chain.nodename, chain)
	}

	chain.Consensus = consensus.NewMolasses(producer, user, psync)
	return nil
}

func (chain *Chain) TrxEnqueue(GroupId string, trx *quorumpb.Trx) error {
	return TrxEnqueue(GroupId, trx)
}

func (chain *Chain) StartSync() error {
	chain_log.Debugf("<%s> StartSync called", chain.groupItem.GroupId)

	//check what kind of sync is needed here
	/*
		//producer try get consensus before start sync block
		if sr.chainCtx.isProducer() {
			syncerrunner_log.Debugf("<%s> producer(owner) node try get latest chain info before start sync", sr.groupId)

			task, err = sr.GetPSyncTask()
			if err != nil {
				return err
			}

		} else {
			//user node start sync directly
			groupMgr_log.Debugf("<%s> user node start epoch (block) sync", sr.groupId)
			task, err = sr.GetNextEpochTask()
			if err != nil {
				return err
			}
		}
	*/

	// since current implementation only support 1 producer (owner), no psync will be prefered
	// all node (except owner) start sync after start up, and finish sync till owner told NO_MORE_BLOCK
	if chain.isOwner() {
		chain_log.Debugf("<%s> Owner no need to sync", chain.groupItem.GroupId)
		return nil
	}

	//TBD check if other sync is ongoing and decide what to do next
	chain.rexSyncer.Start()
	return nil
}

func (chain *Chain) isProducer() bool {
	_, ok := chain.producerPool[chain.groupItem.UserSignPubkey]
	return ok
}

func (chain *Chain) isProducerByPubkey(pubkey string) bool {
	_, ok := chain.producerPool[pubkey]
	return ok
}

func (chain *Chain) isOwnerByPubkey(pubkey string) bool {
	return chain.groupItem.OwnerPubKey == pubkey
}

func (chain *Chain) isOwner() bool {
	return chain.groupItem.OwnerPubKey == chain.groupItem.UserSignPubkey
}

func (chain *Chain) StopSync() {
	chain_log.Debugf("<%s> StopSync called", chain.groupItem.GroupId)
	if chain.rexSyncer != nil {
		chain.rexSyncer.Stop()
	}
}

func (chain *Chain) GetRexSyncerStatus() string {
	status := chain.rexSyncer.GetSyncerStatus()
	statusStr := ""

	//cast status to string
	switch status {
	case IDLE:
		statusStr = "IDLE"
	case SYNCING:
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

func (chain *Chain) GetNextNonce(groupId string, prefix ...string) (nonce uint64, err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	n, err := nodectx.GetDbMgr().GetNextNonce(groupId, nodeprefix)
	return n, err
}

func (chain *Chain) ApplyTrxsFullNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsFullNode called", chain.groupItem.GroupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			chain_log.Debugf("<%s> %s", chain.groupItem.GroupId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> existed, do nothing", chain.groupItem.GroupId, trx.TrxId)
			//nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
			continue
		}

		//new trx, apply it
		chain_log.Debugf("<%s> try apply trx <%s>", chain.groupItem.GroupId, trx.TrxId)

		originalData := trx.Data
		if trx.Type == quorumpb.TrxType_POST && chain.groupItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private Group, encrypted by pgp for all announced Group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(chain.groupItem.GroupId, trx.Data)
			if err != nil {
				//if decrypt error, set trxdata to empty []
				trx.Data = []byte("")
			} else {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}
		} else {
			ciperKey, err := hex.DecodeString(chain.groupItem.CipherKey)
			if err != nil {
				return err
			}

			decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
			if err != nil {
				return err
			}

			trx.Data = decryptData
		}

		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Debugf("<%s> apply POST trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().AddPost(trx, nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.updProducerList()
			chain.updAnnouncedProducerStatus()
			chain.updProducerConfig()
			//chain.UpdConnMgrProducer()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.updUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_APP_CONFIG:
			chain_log.Debugf("<%s> apply APP_CONFIG trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAppConfigTrx(trx, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.groupItem.GroupId, trx.Type.String())
		}

		//set trx data to original(encrypted)
		trx.Data = originalData

		//save original trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}
	return nil
}

func (chain *Chain) ApplyTrxsProducerNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsProducerNode called", chain.groupItem.GroupId)
	for _, trx := range trxs {
		//producer node does not handle APP_CONFIG and POST
		if trx.Type == quorumpb.TrxType_APP_CONFIG || trx.Type == quorumpb.TrxType_POST {
			//chain_log.Infof("Skip TRX %s with type %s", trx.TrxId, trx.Type.String())
			continue
		}

		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			chain_log.Debugf("<%s> %s", chain.groupItem.GroupId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> existed, do nothing", chain.groupItem.GroupId, trx.TrxId)
			//nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
			continue
		}

		originalData := trx.Data
		//decode trx data
		ciperKey, err := hex.DecodeString(chain.groupItem.CipherKey)
		if err != nil {
			return err
		}

		decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
		if err != nil {
			return err
		}

		//set trx.Data to decrypted []byte
		trx.Data = decryptData

		chain_log.Debugf("<%s> apply trx <%s>", chain.groupItem.GroupId, trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.updProducerList()
			chain.updAnnouncedProducerStatus()
			chain.updProducerConfig()
			chain.UpdConnMgrProducer()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.updUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.groupItem.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.groupItem.GroupId, trx.Type)
		}

		trx.Data = originalData

		//save trx to db
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

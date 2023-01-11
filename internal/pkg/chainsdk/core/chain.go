package chain

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
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
	group        *Group
	producerPool map[string]*quorumpb.ProducerItem
	userPool     map[string]*quorumpb.UserItem
	trxFactory   *rumchaindata.TrxFactory
	syncerrunner *SyncerRunner
	chaindata    *ChainData
	Consensus    def.Consensus
	CurrEpoch    int64
	LatestUpdate int64
	//add an atomic EPOCH number
}

func (chain *Chain) NewChain(group *Group) error {
	chain_log.Debugf("<%s> NewChain called", group.Item.GroupId)

	chain.group = group

	//initial TrxFactory
	chain.trxFactory = &rumchaindata.TrxFactory{}
	chain.trxFactory.Init(nodectx.GetNodeCtx().Version, group.Item, group.Nodename, chain)

	//initial Syncer
	chain.syncerrunner = NewSyncerRunner(group.GroupId, group.Nodename, chain, chain)

	//initial chaindata manager
	chain.chaindata = &ChainData{
		nodename:       group.Nodename,
		groupId:        group.GroupId,
		groupCipherKey: group.Item.CipherKey,
		userSignPubkey: group.Item.UserSignPubkey,
		dbmgr:          nodectx.GetDbMgr()}

	//load and initial CurrEpoch/lastUpdate
	currEpoch, lastUpdate, _ := nodectx.GetNodeCtx().GetChainStorage().GetChainInfo()
	chain.SetCurrEpoch(currEpoch)
	chain.SetLastUpdate(lastUpdate)

	return nil
}

// atomic opt for currEpoch
func (chain *Chain) SetCurrEpoch(currEpoch int64) {
	atomic.StoreInt64(&chain.CurrEpoch, currEpoch)
}

func (chain *Chain) IncCurrEpoch() {
	atomic.AddInt64(&chain.CurrEpoch, 1)
}

func (chain *Chain) DecrCurrEpoch() {
	atomic.AddInt64(&chain.CurrEpoch, -1)
}

func (chain *Chain) GetCurrEpoch() int64 {
	return atomic.LoadInt64(&chain.CurrEpoch)
}

// atomic opt for lastUpdate
func (chain *Chain) SetLastUpdate(lastUpdate int64) {
	atomic.StoreInt64(&chain.LatestUpdate, lastUpdate)
}

func (chain *Chain) GetLastUpdate() int64 {
	return atomic.LoadInt64(&chain.LatestUpdate)
}

func (chain *Chain) SaveChainInfoToDb() error {
	chain_log.Debugf("<%s> SaveChainInfoToDb called", chain.group.GroupId)
	chain_log.Debugf("<%s> Current Epoch <%d>, lastUpdate <%d>", chain.group.GroupId, chain.GetCurrEpoch(), chain.GetLastUpdate())
	return nodectx.GetNodeCtx().GetChainStorage().SaveChainInfo(chain.GetCurrEpoch(), chain.GetLastUpdate(), chain.group.GroupId, chain.group.Nodename)
}

func (chain *Chain) GetTrxFactory() chaindef.TrxFactoryIface {
	chain_log.Debugf("<%s> GetTrxFactory called", chain.group.GroupId)
	return chain.trxFactory
}

func (chain *Chain) GetPubqueueIface() chaindef.PublishQueueIface {
	chain_log.Debugf("<%s> GetPubqueueIface called", chain.group.GroupId)
	return GetPubQueueWatcher()
}

// ????
func (chain *Chain) GetConsensus() (string, error) {
	chain_log.Debugf("<%s> GetConsensus called", chain.group.GroupId)
	return chain.syncerrunner.GetConsensus()
}

// PSConn msg handler
func (chain *Chain) HandlePsConnMessage(pkg *quorumpb.Package) error {
	chain_log.Debugf("<%s> HandlePsConnMessage called", chain.group.GroupId)
	var err error
	if pkg.Type == quorumpb.PackageType_BLOCK {
		chain_log.Info("BLOCK msg")
		blk := &quorumpb.Block{}
		err = proto.Unmarshal(pkg.Data, blk)
		if err != nil {
			chain_log.Warning(err.Error())
		} else {
			err = chain.HandleBlockPsConn(blk)
		}
	} else if pkg.Type == quorumpb.PackageType_TRX {
		chain_log.Info("TRX msg")
		trx := &quorumpb.Trx{}
		err = proto.Unmarshal(pkg.Data, trx)
		if err != nil {
			chain_log.Warningf(err.Error())
		} else {
			err = chain.HandleTrxPsConn(trx)
		}
	} else if pkg.Type == quorumpb.PackageType_HBB {
		chain_log.Info("HBB msg")
		hb := &quorumpb.HBMsgv1{}
		err = proto.Unmarshal(pkg.Data, hb)
		if err != nil {
			chain_log.Warningf(err.Error())
		} else {
			err = chain.HandleHBPsConn(hb)
		}
	} else if pkg.Type == quorumpb.PackageType_CONSENSUS {
		chain_log.Info("CONSENSUS msg")
		cm := &quorumpb.ConsensusMsg{}
		err = proto.Unmarshal(pkg.Data, cm)
		if err != nil {
			chain_log.Warnf(err.Error())
		} else {
			err = chain.HandlePSyncConsesusPsConn(cm)
		}
	}

	return err
}

// Handle Trx from PsConn
func (chain *Chain) HandleTrxPsConn(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleTrxPsConn called", chain.group.GroupId)

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
		chain_log.Warningf("<%s> verify Trx failed with err <%s>", chain.group.GroupId, err.Error())
		return fmt.Errorf("verify Trx failed")
	}

	if !verified {
		chain_log.Warningf("<%s> invalid Trx, signature verify failed, sender <%s>", chain.group.GroupId, trx.SenderPubkey)
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
		chain_log.Warningf("<%s> unsupported msg type", chain.group.GroupId)
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func (chain *Chain) producerAddTrx(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> producerAddTrx called", chain.group.GroupId)
	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return nil
	}

	chain.Consensus.Producer().AddTrx(trx)
	return nil
}

// handle BLOCK msg from PSconn
func (chain *Chain) HandleBlockPsConn(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> HandleBlockPsConn called", chain.group.GroupId)

	// all approved producers(owner) should ignore block from psconn (they gonna build block by themselves)
	// when sync, for all node blocks will come from rex channel
	if chain.isProducer() {
		//chain_log.Infof("producer(owner) ignore incoming block from psconn")
		return nil
	}

	//check if block is from approved producer
	if !chain.isProducerByPubkey(block.BookkeepingPubkey) {
		chain_log.Warningf("<%s> received block <%d> from unapproved producer <%s>, reject it", chain.group.Item.GroupId, block.Epoch, block.BookkeepingPubkey)
		return nil
	}

	//for all node run as PRODUCER_NODE but not approved by owner (yet)
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		chain_log.Debugf("<%s> producer node add block", chain.group.GroupId)
		err := chain.Consensus.Producer().AddBlock(block)
		if err != nil {
			chain_log.Warningf("<%s> producer node add block error <%s>", chain.group.GroupId, err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				chain_log.Warningf("<%s> TBD parent not exist, block epoch <%d>, currEpoch <%d>",
					chain.group.GroupId, block.Epoch, chain.GetCurrEpoch())
			}
		}
		return err
	}

	//for all node run as FULLNODE (except owner)
	err := chain.Consensus.User().AddBlock(block)
	if err != nil {
		chain_log.Debugf("<%s> FULLNODE add block error <%s>", chain.group.GroupId, err.Error())
		if err.Error() == "PARENT_NOT_EXIST" {
			chain_log.Infof("<%s> TBD parent not exist, block epoch <%s>, currEpoch <%d>",
				chain.group.GroupId, block.Epoch, chain.GetCurrEpoch())
		}
	}

	return nil
}

// handle HBB msg from PsConn
func (chain *Chain) HandleHBPsConn(hb *quorumpb.HBMsgv1) error {
	chain_log.Debugf("<%s> HandleHBPsConn called", chain.group.GroupId)

	//only producers(owner) need to handle HBB message
	if !chain.isProducer() {
		return nil
	}

	if hb.PayloadType == quorumpb.HBMsgPayloadType_HB_TRX {
		if chain.Consensus.Producer() == nil {
			chain_log.Warningf("<%s> Consensus Producer is null", chain.group.GroupId)
			return nil
		}
		return chain.Consensus.Producer().HandleHBMsg(hb)
	} else if hb.PayloadType == quorumpb.HBMsgPayloadType_HB_PSYNC {
		if chain.Consensus.PSync() == nil {
			chain_log.Warningf("<%s> Consensus PSync is null", chain.group.GroupId)
			return nil
		}
		return chain.Consensus.PSync().HandleHBMsg(hb)
	}

	return fmt.Errorf("unknown hbmsg type %s", hb.PayloadType.String())
}

// handle psync consensus req from PsConn
func (chain *Chain) HandlePSyncConsesusPsConn(c *quorumpb.ConsensusMsg) error {
	chain_log.Debugf("<%s> HandlePSyncConsesusReqPsConn called", chain.group.GroupId)

	//only producers(owner) need to handle Consensus msg
	if !chain.isProducer() {
		return nil
	}

	if chain.Consensus.PSync() == nil {
		chain_log.Warningf("<%s> Consensus PSync is null", chain.group.GroupId)
		return nil
	}

	d := &quorumpb.ConsensusMsg{
		GroupId:      c.GroupId,
		SessionId:    c.SessionId,
		MsgType:      c.MsgType,
		Payload:      c.Payload,
		SenderPubkey: c.SenderPubkey,
		TimeStamp:    c.TimeStamp,
	}

	db, err := proto.Marshal(d)
	if err != nil {
		return err
	}

	//check hash
	dhash := localcrypto.Hash(db)
	if res := bytes.Compare(c.MsgHash, dhash); res != 0 {
		return fmt.Errorf("msg hash mismatch")
	}

	//check signature
	bytespubkey, err := base64.RawURLEncoding.DecodeString(c.SenderPubkey)
	if err != nil {
		return err
	}
	ethpbukey, err := ethcrypto.DecompressPubkey(bytespubkey)
	if err == nil {
		ks := localcrypto.GetKeystore()
		r := ks.EthVerifySign(c.MsgHash, c.SenderSign, ethpbukey)
		if !r {
			return fmt.Errorf("verify signature failed")
		}
	} else {
		return err
	}

	if c.MsgType == quorumpb.ConsensusType_REQ {
		if !chain.isProducerByPubkey(c.SenderPubkey) {
			chain_log.Warningf("consensusReq from non producer node <%s>, ignore", c.GroupId)
			return nil
		}
		//let psync handle the req
		return chain.Consensus.PSync().AddConsensusReq(c)
	} else if c.MsgType == quorumpb.ConsensusType_RESP {
		//check if the resp is from myself
		if len(chain.producerPool) != 1 && chain.group.Item.UserSignPubkey == c.SenderPubkey {
			chain_log.Debugf("multiple producer exist, session <%s> consensusResp from myself, ignore", c.SessionId)
			return nil
		}

		//check if psync result with same session_id exist
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsPSyncSessionExist(chain.group.GroupId, c.SessionId)
		if err != nil {
			return err
		}

		if isExist {
			chain_log.Debugf("Session <%s> is handled, ignore", c.SessionId)
			return nil
		}

		//verify response
		resp := &quorumpb.ConsensusResp{}
		err = proto.Unmarshal(c.Payload, resp)
		if err != nil {
			return err
		}

		ok, err := chain.verifyProducer(c.SenderPubkey, resp)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("invalid consensusResp from producer <%s>", c.SenderPubkey)
		}

		return chain.handlePSyncResp(c.SessionId, resp)
	} else {
		return fmt.Errorf("unknown msgType %s", c.MsgType)
	}
}

func (chain *Chain) handlePSyncResp(sessionId string, resp *quorumpb.ConsensusResp) error {
	chain_log.Debugf("<%s> handlePSyncResp called, SessionId <%s>", chain.group.GroupId, sessionId)

	//check if the resp is what gsync expected
	taskId, taskType, _, err := chain.syncerrunner.GetCurrentSyncTask()
	if err == rumerrors.ErrNoTaskWait || taskType != ConsensusSync || taskId != sessionId {
		//not the expected consensus resp
		return rumerrors.ErrConsusMismatch
	}

	savedResp, err := nodectx.GetNodeCtx().GetChainStorage().GetCurrentPSyncSession(chain.group.GroupId)
	if err != nil {
		return err
	}

	//just in case
	if len(savedResp) != 1 {
		chain_log.Warningf("<%s> get <%d> saved psync resp msg (should be 1), something goes wrong", len(savedResp), chain.group.GroupId)
		return fmt.Errorf("psync resp msg mismatch, something goes wrong")
	}

	respItem := savedResp[0]
	if respItem.CurChainEpoch > resp.CurChainEpoch {
		chain_log.Debugf("resp from old epoch, do nothing, ignore")
		return fmt.Errorf("resp from old epoch, ignore")
	}

	//save ConsensusResp
	nodectx.GetNodeCtx().GetChainStorage().UpdPSyncResp(chain.group.GroupId, sessionId, resp)

	//TBD check and update producer according to psync resp
	/*
		trx, _, err := nodectx.GetNodeCtx().GetChainStorage().GetTrx(chain.groupId, resp.ProducerProof.TrxId, sdef.Chain, chain.nodename)
		if err != nil && trx != nil {
			chain_log.Debugf("No need to upgrade producer list")
		} else {
			//TBD update producers list and regerate all consensus
			// user
			// producer
			// psync
		}
	*/

	if resp.CurChainEpoch == chain.GetCurrEpoch() {
		chain_log.Debugf("node local epoch == current chain epoch, No need to sync")
		chain.syncerrunner.UpdateConsensusResult(sessionId, uint(SyncDone))
	} else {
		chain.syncerrunner.UpdateConsensusResult(sessionId, uint(ContinueGetEpoch))
	}

	return nil
}

func (chain *Chain) verifyProducer(senderPubkey string, resp *quorumpb.ConsensusResp) (bool, error) {
	chain_log.Debugf("<%s> verifyProducer called", chain.group.GroupId)

	//TBD, verify signature for ConsensusResp

	//consensusResp from owner, trust it anyway
	if senderPubkey == chain.group.Item.OwnerPubKey {
		return true, nil
	}

	//conosensusResp form other producer, in this resp,
	//no other producers approved (owner works as the only group producer)
	//no related trx to be verified
	if len(resp.CurProducer.Producers) == 1 && resp.CurProducer.Producers[0] == chain.group.Item.OwnerPubKey {
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
	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
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

func (chain *Chain) HandleConsesusRex(c *quorumpb.ConsensusMsg) error {
	return nil
}

// handler trx from rex (for sync only)
func (chain *Chain) HandleTrxRex(trx *quorumpb.Trx, s network.Stream) error {
	chain_log.Debugf("<%s> HandleTrxRex called", chain.group.GroupId)
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Warningf("HandleTrxRex called, Trx Version mismatch, trxid <%s>: <%s> vs <%s>", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return fmt.Errorf("trx Version mismatch")
	}

	verified, err := rumchaindata.VerifyTrx(trx)
	if err != nil {
		chain_log.Warningf("<%s> verify Trx failed with err <%s>", chain.group.GroupId, err.Error())
		return fmt.Errorf("verify Trx failed")
	}

	if !verified {
		chain_log.Warnf("<%s> Invalid Trx, signature verify failed, sender <%s>", chain.group.GroupId, trx.SenderPubkey)
		return fmt.Errorf("invalid Trx")
	}

	if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
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
	chain_log.Debugf("<%s> HandleBlockRex called", chain.group.GroupId)
	return nil
}

// unused
func (chain *Chain) HandleHBRex(hb *quorumpb.HBMsgv1) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.group.GroupId)
	return nil
}

func (chain *Chain) handleReqBlocks(trx *quorumpb.Trx, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlocks called", chain.group.GroupId)
	requester, fromEpoch, blkReqs, blocks, result, err := chain.chaindata.GetReqBlocks(trx)
	if err != nil {
		return err
	}

	chain_log.Debugf("<%s> send REQ_BLOCKS_RESP", chain.group.GroupId)
	chain_log.Debugf("-- requester <%s>, fromEpoch <%d>, request <%d>", requester, fromEpoch, blkReqs)
	chain_log.Debugf("-- send fromEpoch <%d>, total <%d> blocks, status <%s>", fromEpoch, len(blocks), result.String())

	trx, err = chain.trxFactory.GetReqBlocksRespTrx("", chain.group.GroupId, requester, blkReqs, fromEpoch, blocks, result)
	if err != nil {
		return err
	}

	if cmgr, err := conn.GetConn().GetConnMgr(chain.group.GroupId); err != nil {
		return err
	} else {
		return cmgr.SendRespTrxRex(trx, s)
	}
}

func (chain *Chain) handleReqBlockResp(trx *quorumpb.Trx) { //taskId,error
	chain_log.Debugf("<%s> HandleReqBlockResp called", chain.group.GroupId)

	//decode resp
	var err error
	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
	if err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, err.Error())
		return
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, err.Error())
		return
	}

	reqBlockResp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(decryptData, reqBlockResp); err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, err.Error())
		return
	}

	//if not asked by me, ignore it
	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		//chain_log.Debugf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, rumerrors.ErrSenderMismatch.Error())
		return
	}

	//check trx sender
	if trx.SenderPubkey != reqBlockResp.ProviderPubkey {
		chain_log.Debugf("<%s> HandleReqBlockResp - Trx Sender/blocks providers mismatch <%s>", chain.group.GroupId)
		return
	}

	gsyncerTaskId, gsyncerTaskType, _, err := chain.syncerrunner.GetCurrentSyncTask()
	if err == rumerrors.ErrNoTaskWait {
		chain_log.Debugf("<%s> HandleReqBlockResp - no task waiting", chain.group.GroupId)
		return
	}

	if gsyncerTaskType != GetEpoch {
		chain_log.Debugf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, rumerrors.ErrSyncerStatus.Error())
		return
	}

	//get epoch by using taskId
	epochWaiting, err := strconv.ParseInt(gsyncerTaskId, 10, 64)
	if err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, err.Error())
		return
	}

	//check if the epoch is what we are waiting for
	if reqBlockResp.FromEpoch != epochWaiting {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, rumerrors.ErrEpochMismatch)
		return
	}

	chain_log.Debugf("- Receive valid reqBlockResp, provider <%s> result <%s> from epoch <%d> total blocks provided <%d>",
		reqBlockResp.ProviderPubkey,
		reqBlockResp.Result.String(),
		reqBlockResp.FromEpoch,
		len(reqBlockResp.Blocks.Blocks))

	isFromProducer := chain.isProducerByPubkey(reqBlockResp.ProviderPubkey)

	switch reqBlockResp.Result {
	case quorumpb.ReqBlkResult_BLOCK_NOT_FOUND:
		//user node say BLOCK_NOT_FOUND, ignore
		if !isFromProducer {
			chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_NOT_FOUND from user node <%s>, ignore", chain.group.GroupId, reqBlockResp.ProviderPubkey)
			return
		}

		//TBD, stop only when received BLOCK_NOT_FOUND from F + 1 producers, otherwise continue sync
		chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_NOT_FOUND from producer node <%s>, process it", chain.group.GroupId, reqBlockResp.ProviderPubkey)
		taskId := strconv.Itoa(int(reqBlockResp.FromEpoch))
		chain.syncerrunner.UpdateGetEpochResult(taskId, uint(SyncDone))
		return

	case quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP:
		chain.applyBlocks(reqBlockResp.Blocks.Blocks)
		if !isFromProducer {
			chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP_ON_TOP from user node <%s>, apply all blocks and  ignore ON_TOP", chain.group.GroupId, reqBlockResp.ProviderPubkey)
			return
		}

		chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP_ON_TOP from producer node <%s>, process it", chain.group.GroupId, reqBlockResp.ProviderPubkey)
		//ignore on_top msg, run another round of sync, till get F + 1 BLOCK_NOT_FOUND from producers
		chain.syncerrunner.UpdateGetEpochResult(gsyncerTaskId, uint(ContinueGetEpoch))
		return
	case quorumpb.ReqBlkResult_BLOCK_IN_RESP:
		chain_log.Debugf("<%s> HandleReqBlockResp - receive BLOCK_IN_RESP from node <%s>, apply all blocks", chain.group.GroupId, reqBlockResp.ProviderPubkey)
		chain.applyBlocks(reqBlockResp.Blocks.Blocks)
		chain.syncerrunner.UpdateGetEpochResult(gsyncerTaskId, uint(ContinueGetEpoch))
		break
	}
}

func (chain *Chain) applyBlocks(blocks []*quorumpb.Block) error {
	//PRODUCER_NODE add SYNC
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		for _, block := range blocks {
			err := chain.Consensus.Producer().AddBlock(block)
			if err != nil {
				chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, err.Error())
				return err
			}
		}
	}

	//FULLNODE (include owner) Add synced Block
	for _, block := range blocks {
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.group.GroupId, err.Error())
			return err
		}
	}
	return nil
}

func (chain *Chain) UpdConnMgrProducer() {
	chain_log.Debugf("<%s> UpdConnMgrProducer called", chain.group.GroupId)
	connMgr, _ := conn.GetConn().GetConnMgr(chain.group.GroupId)

	var producerspubkey []string
	for key, _ := range chain.producerPool {
		producerspubkey = append(producerspubkey, key)
	}

	connMgr.UpdProducers(producerspubkey)
}

func (chain *Chain) updProducerList() {
	chain_log.Debugf("<%s> UpdProducerList called", chain.group.GroupId)
	//create and load group producer pool
	chain.producerPool = make(map[string]*quorumpb.ProducerItem)
	producers, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(chain.group.Item.GroupId, chain.group.Nodename)

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
		if item.ProducerPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> load producer <%s%s>", chain.group.GroupId, item.ProducerPubkey, ownerPrefix)
	}
}

func (chain *Chain) updAnnouncedProducerStatus() {
	chain_log.Debugf("<%s> updAnnouncedProducerStatus called", chain.group.GroupId)

	//update announced producer result
	announcedProducers, _ := nodectx.GetNodeCtx().GetChainStorage().GetAnnounceProducersByGroup(chain.group.Item.GroupId, chain.group.Nodename)
	for _, item := range announcedProducers {
		_, ok := chain.producerPool[item.SignPubkey]
		err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounceResult(quorumpb.AnnounceType_AS_PRODUCER, chain.group.Item.GroupId, item.SignPubkey, ok, chain.group.Nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.group.GroupId, err.Error())
		}
	}
}

func (chain *Chain) updProducerConfig() {
	chain_log.Debugf("<%s> updProducerConfig called", chain.group.GroupId)
	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return
	}

	//recreate producer BFT config
	chain.Consensus.Producer().RecreateBft()
}

func (chain *Chain) updUserList() {
	chain_log.Debugf("<%s> updUserList called", chain.group.GroupId)

	//create and load group user pool
	chain.userPool = make(map[string]*quorumpb.UserItem)
	users, _ := nodectx.GetNodeCtx().GetChainStorage().GetUsers(chain.group.Item.GroupId, chain.group.Nodename)
	for _, item := range users {
		chain.userPool[item.UserPubkey] = item
		ownerPrefix := "(user)"
		if item.UserPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> Load Users <%s_%s>", chain.group.GroupId, item.UserPubkey, ownerPrefix)
	}

	//update announced User result
	announcedUsers, _ := nodectx.GetNodeCtx().GetChainStorage().GetAnnounceUsersByGroup(chain.group.Item.GroupId, chain.group.Nodename)
	for _, item := range announcedUsers {
		_, ok := chain.userPool[item.SignPubkey]
		err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounceResult(quorumpb.AnnounceType_AS_USER,
			chain.group.Item.GroupId,
			item.SignPubkey,
			ok,
			chain.group.Nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.group.GroupId, err.Error())
		}
	}
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

func (chain *Chain) CreateConsensus() error {
	chain_log.Debugf("<%s> CreateConsensus called", chain.group.GroupId)

	var user def.User
	var producer def.Producer
	var psync def.PSync

	var shouldCreateUser, shouldCreateProducer, shouldCreatePSyncer bool

	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		shouldCreateProducer = true
		shouldCreateUser = false
		shouldCreatePSyncer = true
	} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
		//check if I am owner of the group
		if chain.group.Item.UserSignPubkey == chain.group.Item.OwnerPubKey {
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
		chain_log.Infof("<%s> Create and initial molasses producer", chain.group.GroupId)
		producer = &consensus.MolassesProducer{}
		producer.NewProducer(chain.group.Item, chain.group.Nodename, chain)
	}

	if shouldCreateUser {
		chain_log.Infof("<%s> Create and initial molasses user", chain.group.GroupId)
		user = &consensus.MolassesUser{}
		user.NewUser(chain.group.Item, chain.group.Nodename, chain)
	}

	if shouldCreatePSyncer {
		chain_log.Infof("<%s> Create and initial molasses psyncer", chain.group.GroupId)
		psync = &consensus.MolassesPSync{}
		psync.NewPSync(chain.group.Item, chain.group.Nodename, chain)
	}

	chain.Consensus = consensus.NewMolasses(producer, user, psync)
	return nil
}

func (chain *Chain) TrxEnqueue(groupId string, trx *quorumpb.Trx) error {
	return TrxEnqueue(groupId, trx)
}

func (chain *Chain) StartSync() error {
	chain_log.Debugf("<%s> StartSync called", chain.group.GroupId)
	return chain.syncerrunner.Start()
}

func (chain *Chain) isProducer() bool {
	_, ok := chain.group.ChainCtx.producerPool[chain.group.Item.UserSignPubkey]
	return ok
}

func (chain *Chain) isProducerByPubkey(pubkey string) bool {
	_, ok := chain.group.ChainCtx.producerPool[pubkey]
	return ok
}

func (chain *Chain) StopSync() {
	chain_log.Debugf("<%s> StopSync called", chain.group.GroupId)
	if chain.syncerrunner != nil {
		chain.syncerrunner.Stop()
	}
}

func (chain *Chain) GetSyncerStatus() int8 {
	return chain.syncerrunner.gsyncer.Status
}

func (chain *Chain) IsSyncerIdle() bool {
	chain_log.Debugf("IsSyncerIdle called, groupId <%s>", chain.group.GroupId)
	if chain.syncerrunner.gsyncer.Status == SYNCING_FORWARD ||
		chain.syncerrunner.gsyncer.Status == LOCAL_SYNCING ||
		chain.syncerrunner.gsyncer.Status == CONSENSUS_SYNC ||
		chain.syncerrunner.gsyncer.Status == SYNC_FAILED {
		chain_log.Debugf("<%s> gsyncer is busy, status: <%d>", chain.group.GroupId, chain.syncerrunner.gsyncer.Status)
		return true
	}
	chain_log.Debugf("<%s> syncer is IDLE", chain.group.GroupId)
	return false
}

func (chain *Chain) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	n, err := nodectx.GetDbMgr().GetNextNouce(groupId, nodeprefix)
	return n, err
}

func (chain *Chain) ApplyTrxsFullNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsFullNode called", chain.group.GroupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			chain_log.Debugf("<%s> %s", chain.group.GroupId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> existed, do nothing", chain.group.GroupId, trx.TrxId)
			//nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
			continue
		}

		//new trx, apply it
		chain_log.Debugf("<%s> try apply trx <%s>", chain.group.GroupId, trx.TrxId)

		originalData := trx.Data
		if trx.Type == quorumpb.TrxType_POST && chain.group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(chain.group.GroupId, trx.Data)
			if err != nil {
				//if decrypt error, set trxdata to empty []
				trx.Data = []byte("")
			} else {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}
		} else {
			ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
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
			chain_log.Debugf("<%s> apply POST trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().AddPost(trx, nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.updProducerList()
			chain.updAnnouncedProducerStatus()
			chain.updProducerConfig()
			//chain.UpdConnMgrProducer()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.updUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_APP_CONFIG:
			chain_log.Debugf("<%s> apply APP_CONFIG trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAppConfigTrx(trx, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.group.GroupId, trx.Type.String())
		}

		//set trx data to original(encrypted)
		trx.Data = originalData

		//save original trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}
	return nil
}

func (chain *Chain) ApplyTrxsProducerNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsProducerNode called", chain.group.GroupId)
	for _, trx := range trxs {
		//producer node does not handle APP_CONFIG and POST
		if trx.Type == quorumpb.TrxType_APP_CONFIG || trx.Type == quorumpb.TrxType_POST {
			//chain_log.Infof("Skip TRX %s with type %s", trx.TrxId, trx.Type.String())
			continue
		}

		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.GroupId, trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			chain_log.Debugf("<%s> %s", chain.group.GroupId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> existed, do nothing", chain.group.GroupId, trx.TrxId)
			//nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
			continue
		}

		originalData := trx.Data
		//decode trx data
		ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
		if err != nil {
			return err
		}

		decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
		if err != nil {
			return err
		}

		//set trx.Data to decrypted []byte
		trx.Data = decryptData

		chain_log.Debugf("<%s> apply trx <%s>", chain.group.GroupId, trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.updProducerList()
			chain.updAnnouncedProducerStatus()
			chain.updProducerConfig()
			chain.UpdConnMgrProducer()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.updUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.group.GroupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.group.GroupId, trx.Type)
		}

		trx.Data = originalData

		//save trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}

	return nil
}

//local sync
//TODO
//func (chain *Chain) SyncLocalBlock() error {
//	startFrom := chain.group.Item.HighestBlockId
//	for {
//		subblocks, err := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(chain.group.Item.HighestBlockId, chain.nodename)
//		if err != nil {
//			chain_log.Debugf("<%s> GetSubBlock failed <%s>", chain.groupId, err.Error())
//			return err
//		}
//		if len(subblocks) > 0 {
//			for _, block := range subblocks {
//				err := chain.AddLocalBlock(block)
//				if err != nil {
//					chain_log.Debugf("<%s> AddLocalBlock failed <%s>", chain.groupId, err.Error())
//					break // for range subblocks
//				}
//			}
//		} else {
//			chain_log.Debugf("<%s> No more local blocks", chain.groupId)
//			return nil
//		}
//		topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(chain.group.Item.HighestBlockId, false, chain.nodename)
//		if err != nil {
//			chain_log.Debugf("<%s> Get Top Block failed <%s>", chain.groupId, err.Error())
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
//	chain_log.Debugf("<%s> AddLocalBlock called", chain.groupId)
//	signpkey, err := localcrypto.Libp2pPubkeyToEthBase64(chain.group.Item.UserSignPubkey)
//	if err != nil && signpkey == "" {
//		chain_log.Warnf("<%s> Pubkey err <%s>", chain.groupId, err)
//	}
//
//	_, producer := chain.ProducerPool[signpkey]
//
//	if producer {
//		chain_log.Debugf("<%s> PRODUCER ADD LOCAL BLOCK <%d>", chain.groupId, block.Epoch)
//		err := chain.AddBlock(block)
//		if err != nil {
//			chain_log.Infof(err.Error())
//		}
//	} else {
//		chain_log.Debugf("<%s> USER ADD LOCAL BLOCK <%d>", chain.groupId, block.Epoch)
//		err := chain.Consensus.User().AddBlock(block)
//		if err != nil {
//			chain_log.Infof(err.Error())
//		}
//	}
//	return nil
//}

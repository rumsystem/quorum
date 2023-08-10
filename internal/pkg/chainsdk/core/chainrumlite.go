package chain

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var chain_log = logging.Logger("chain")
var DEFAULT_PROPOSE_TRX_INTERVAL = 1000 //ms

// rum lite
type ChainRumLite struct {
	groupItem       *quorumpb.GroupItemRumLite
	nodename        string
	trxFactory      *rumchaindata.TrxFactoryRumLite
	syncerWhiteList map[string]bool
	rexSyncer       *RexLiteSyncer
	chainData       *ChainDataRumLite
	LastUpdate      int64
	ChainCtx        context.Context
	CtxCancelFunc   context.CancelFunc
	Consensus       def.ConsensusRumLite
}

func (chain *ChainRumLite) NewChainRumLite(item *quorumpb.GroupItemRumLite, nodename string) error {
	chain_log.Debugf("<%s> NewChainRumLite called", item.GroupId)

	chain.groupItem = item
	chain.nodename = nodename

	//initial TrxFactory
	chain.trxFactory = &rumchaindata.TrxFactoryRumLite{}
	chain.trxFactory.Init(nodectx.GetNodeCtx().Version, chain.groupItem, chain.nodename)

	//initial sync white list
	chain.syncerWhiteList = make(map[string]bool)

	//initial Syncer
	//chain.rexSyncer = NewRexLiteSyncer(chain.ChainCtx, chain.groupItem, chain.nodename, chain, chain)

	//initial chaindata manager
	chain.chainData = &ChainDataRumLite{
		nodename:      chain.nodename,
		groupId:       chain.groupItem.GroupId,
		cipherKey:     chain.groupItem.CipherKey,
		trxSignPubkey: chain.groupItem.TrxSignPubkey,
		dbmgr:         nodectx.GetDbMgr(),
	}

	//create context with cancel function, chainCtx will be ctx parent of all underlay components
	chain.ChainCtx, chain.CtxCancelFunc = context.WithCancel(nodectx.GetNodeCtx().Ctx)

	chain_log.Debugf("<%s> NewChain done", chain.groupItem.GroupId)
	return nil
}

func (chain *ChainRumLite) GetTrxFactory() chaindef.TrxFactoryIfaceRumLite {
	chain_log.Debugf("<%s> GetTrxFactory called", chain.groupItem.GroupId)
	return chain.trxFactory
}

// PSConn msg handler
func (chain *ChainRumLite) HandlePsConnMessage(pkg *quorumpb.Package) error {
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
func (chain *ChainRumLite) HandleTrxPsConn(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleTrxPsConn called", chain.groupItem.GroupId)

	//TBD check if I have the chain producer key

	//check if trx version match
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Warningf("trx Version mismatch trx_id <%s>: <%s> vs <%s>", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return fmt.Errorf("trx Version mismatch")
	}

	// decompress trx data
	content := new(bytes.Buffer)
	if err := utils.Decompress(bytes.NewReader(trx.Data), content); err != nil {
		chain_log.Errorf("utils.Decompress failed: %s", err)
		return fmt.Errorf("utils.Decompress failed: %s", err)
	}
	trx.Data = content.Bytes()

	//verify trx
	verified, err := rumchaindata.VerifyTrx(trx)
	if err != nil {
		chain_log.Warningf("<%s> verify Trx failed with err <%s>", chain.groupItem.GroupId, err.Error())
		return fmt.Errorf("verify trx failed")
	}
	if !verified {
		chain_log.Warningf("<%s> invalid Trx, signature verify failed, sender <%s>", chain.groupItem.GroupId, trx.SenderPubkey)
		return fmt.Errorf("invalid trx, signature verify failed")
	}

	//handle trx
	switch trx.Type {
	case
		quorumpb.TrxType_POST,
		quorumpb.TrxType_UPD_SYNCER,
		quorumpb.TrxType_CHAIN_CONFIG,
		quorumpb.TrxType_APP_CONFIG,
		quorumpb.TrxType_FORK:
		chain.Consensus.Producer().AddTrxToTxBuffer(trx)
	default:
		chain_log.Warningf("<%s> unsupported msg type", chain.groupItem.GroupId)
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func (chain *ChainRumLite) HandleBftMsgPsConn(msg *quorumpb.BftMsg) error {
	chain_log.Debugf("<%s> HandleHBPTPsConn called", chain.groupItem.GroupId)

	//TBD check if I have the chain producer key

	return chain.Consensus.Producer().HandleBftMsg(msg)
}

func (chain *ChainRumLite) HandleBroadcastMsgPsConn(brd *quorumpb.BroadcastMsg) error {
	chain_log.Debugf("<%s> HandleGroupBroadcastPsConn called", chain.groupItem.GroupId)
	return nil
}

// handler SyncMsg from rex
func (chain *ChainRumLite) HandleSyncMsgRex(syncMsg *quorumpb.SyncMsg, s network.Stream) error {
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

func (chain *ChainRumLite) handleReqBlockRex(syncMsg *quorumpb.SyncMsg, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlocks called", chain.groupItem.GroupId)
	//unmarshall req
	req := &quorumpb.ReqBlock{}
	err := proto.Unmarshal(syncMsg.Data, req)
	if err != nil {
		chain_log.Warningf("<%s> handleReqBlocksRex error <%s>", chain.groupItem.GroupId, err.Error())
		return err
	}

	//do nothing is req is from myself
	if req.ReqPubkey == chain.groupItem.TrxSignPubkey {
		return nil
	}

	if !chain.canSync(req) {
		chain_log.Debugf("<%s> requester <%s> can not sync, ignore REQ_BLOCK", chain.groupItem.GroupId, req.ReqPubkey)
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
	blocks, result, err := chain.chainData.GetReqBlocks(req)
	if err != nil {
		return err
	}

	chain_log.Debugf("<%s> send REQ_BLOCKS_RESP", chain.groupItem.GroupId)
	chain_log.Debugf("-- requester <%s>, from Block <%d>, request <%d> blocks", req.ReqPubkey, req.FromBlock, req.BlksRequested)
	chain_log.Debugf("-- send fromBlock <%d>, total <%d> blocks, status <%s>", req.FromBlock, len(blocks), result.String())

	resp, err := rumchaindata.GetReqBlocksRespMsg("", req, chain.groupItem.TrxSignPubkey, blocks, result)

	if err != nil {
		return err
	}

	if cmgr, err := conn.GetConn().GetConnMgr(chain.groupItem.GroupId); err != nil {
		return err
	} else {
		return cmgr.SendSyncRespMsgRex(resp, s)
	}
}

// check if syncer is allowed
func (chain *ChainRumLite) canSync(req *quorumpb.ReqBlock) bool {
	if chain.groupItem.SyncType == quorumpb.GroupSyncType_PUBLIC_SYNC {
		return true
	}

	return chain.syncerWhiteList[req.ReqPubkey]
}

func (chain *ChainRumLite) handleReqBlockRespRex(syncMsg *quorumpb.SyncMsg) error {
	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupItem.GroupId)

	resp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(syncMsg.Data, resp); err != nil {
		chain_log.Warningf("<%s> HandleReqBlockResp error <%s>", chain.groupItem.GroupId, err.Error())
		return err
	}

	//if not asked by me, ignore it
	if resp.RequesterPubkey != chain.groupItem.TrxSignPubkey {
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

// handle block msg from PSconn (new produced block)
func (chain *ChainRumLite) HandleBlockPsConn(block *quorumpb.BlockRumLite) error {
	chain_log.Debugf("<%s> HandleBlockPsConn called", chain.groupItem.GroupId)
	/*

		//check if block is from a valid group producer
		if !chain.IsProducerByPubkey(block.ProducerPubkey) {
			chain_log.Warningf("<%s> received blockid <%d> from unknown producer <%s>, reject it", chain.groupItem.GroupId, block.BlockId, block.ProducerPubkey)
			return nil
		}

		if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
			chain_log.Debugf("<%s> producer node add block", chain.groupItem.GroupId)
			err := chain.Consensus.Producer().AddBlock(block)
			if err != nil {
				chain_log.Warningf("<%s> producer node add block error <%s>", chain.groupItem.GroupId, err.Error())
				if err.Error() == "PARENT_NOT_EXIST" {
					chain_log.Debugf("<%s> announced producer add block, parent not exist, blockId <%d>, currBlockId <%d>, wait syncing",
						chain.groupItem.GroupId, block.BlockId, chain.GetCurrBlockId())
				}
			}
			return err
		}

		//Fullnode at block
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Debugf("<%s> FULLNODE add block error <%s>", chain.groupItem.GroupId, err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				chain_log.Infof("<%s> block parent not exist, blockId <%s>, currBlockId <%d>, wait syncing",
					chain.groupItem.GroupId, block.BlockId, chain.GetCurrBlockId())
			}
		}
	*/

	return nil
}

func (chain *ChainRumLite) StartSync() error {
	chain_log.Debugf("<%s> StartSync called", chain.groupItem.GroupId)
	chain.rexSyncer.Start()
	return nil
}

func (chain *ChainRumLite) StopSync() {
	chain_log.Debugf("<%s> StopSync called", chain.groupItem.GroupId)
	if chain.rexSyncer != nil {
		chain.rexSyncer.Stop()
	}
}

func (chain *ChainRumLite) GetBlockFromDSCache(groupId string, blockId uint64, prefix ...string) (*quorumpb.Block, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetBlockFromDSCache(groupId, blockId, chain.nodename)
}

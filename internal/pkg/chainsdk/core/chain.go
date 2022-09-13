package chain

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/consensus"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var chain_log = logging.Logger("chain")

type GroupProducer struct {
	ProducerPubkey   string
	ProducerPriority int8
}

type Chain struct {
	nodename           string
	groupId            string
	group              *Group
	userChannelId      string
	producerChannelId  string
	syncChannelId      string
	ProducerPool       map[string]*quorumpb.ProducerItem
	userPool           map[string]*quorumpb.UserItem
	peerIdPool         map[string]string
	chaindata          *ChainData
	Consensus          def.Consensus
	producerChannTimer *time.Timer
	ProviderPeerIdPool map[string]string
	trxFactory         *rumchaindata.TrxFactory
	//request add
	//rexresultmap       utils.TTLMap
	syncerrunner *SyncerRunner
}

func (chain *Chain) ChainInit(group *Group) error {
	chain_log.Debugf("<%s> Init called", group.Item.GroupId)
	chain.group = group

	chain.nodename = nodectx.GetNodeCtx().Name
	chain.groupId = group.Item.GroupId
	chain.chaindata = &ChainData{nodename: chain.nodename, groupId: group.Item.GroupId, groupCipherKey: group.Item.CipherKey, userSignPubkey: group.Item.UserSignPubkey, dbmgr: nodectx.GetDbMgr()}

	chain.trxFactory = &rumchaindata.TrxFactory{}
	chain.trxFactory.Init(nodectx.GetNodeCtx().Version, group.Item, chain.nodename, chain)

	//set rex result map TTL to 30 second. (may not need this map, just check received block will be good?)
	//chain.rexresultmap = NewTTLMap(time.Duration(30) * time.Second)
	chain.syncerrunner = NewSyncerRunner(group, chain, chain.nodename)

	chain_log.Infof("<%s> chainctx initialed", chain.groupId)
	return nil
}

func (chain *Chain) GetNodeName() string {
	return chain.nodename
}

func (chain *Chain) SetRumExchangeTestMode() {
	chain.syncerrunner.SetRumExchangeTestMode()
}

func (chain *Chain) GetChainSyncIface() chaindef.ChainDataSyncIface {
	return chain
}

func (chain *Chain) GetTrxFactory() chaindef.TrxFactoryIface {
	return chain.trxFactory
}

//func (chain *Chain) GetSyncer() *Syncer {
//	return chain.syncer
//}

func (chain *Chain) GetPubqueueIface() chaindef.PublishQueueIface {
	return GetPubQueueWatcher()
}

func (chain *Chain) UpdChainInfo(height int64, blockId string) error {
	chain_log.Debugf("<%s> UpdChainInfo called", chain.groupId)
	chain.group.Item.HighestHeight = height
	chain.group.Item.HighestBlockId = blockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	chain_log.Infof("<%s> Chain Info updated %d, %v", chain.group.Item.GroupId, height, blockId)
	return nodectx.GetNodeCtx().GetChainStorage().UpdGroup(chain.group.Item)
}

func (chain *Chain) HandleTrxRex(trx *quorumpb.Trx, s network.Stream) error {
	chain_log.Debugf("<%s> HandleTrxRex called", chain.groupId)
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Errorf("HandleTrxRex called, Trx Version mismatch %s: %s vs %s", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return errors.New("Trx Version mismatch")
	}

	//Rex Channel only support the following trx type
	switch trx.Type {
	case quorumpb.TrxType_REQ_BLOCK_FORWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockForward(trx, conn.RumExchange, s)
	case quorumpb.TrxType_REQ_BLOCK_BACKWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockBackward(trx, conn.RumExchange, s)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		//use gsyncer, current blockid as taskid
		//!!! don't add repeat trx result to the gsyncer
		chain.syncerrunner.AddTrxToSyncerQueue(trx)
	default:
		//do nothing
	}

	return nil
}

func (chain *Chain) HandleTrxPsConn(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleTrxPsConn called", chain.groupId)
	if trx.Version != nodectx.GetNodeCtx().Version {
		chain_log.Errorf("HandleTrxPsConn called, Trx Version mismatch %s: %s vs %s", trx.TrxId, trx.Version, nodectx.GetNodeCtx().Version)
		return errors.New("Trx Version mismatch")
	}

	verified, err := rumchaindata.VerifyTrx(trx)
	if err != nil {
		chain_log.Warnf("<%s> verify Trx failed with err <%s>", chain.groupId, err.Error())
		return errors.New("Verify Trx failed")
	}

	if !verified {
		chain_log.Warnf("<%s> Invalid Trx, signature verify failed, sender %s", chain.groupId, trx.SenderPubkey)
		return errors.New("Invalid Trx")
	}
	chain_log.Debugf("<%s> HandleTrxPsConn this trx type: %s trxid: %s", chain.groupId, trx.Type, trx.TrxId)

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
		chain.handleReqBlockForward(trx, conn.PubSub, nil)
	case quorumpb.TrxType_REQ_BLOCK_BACKWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockBackward(trx, conn.PubSub, nil)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		//use current blockid as taskid
		chain.syncerrunner.AddTrxToSyncerQueue(trx)
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

func (chain *Chain) HandleBlockRex(block *quorumpb.Block, s network.Stream) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.groupId)
	return nil
}

func (chain *Chain) HandleSnapshotPsConn(snapshot *quorumpb.Snapshot) error {
	chain_log.Debugf("<%s> HandleSnapshotPsConn called", chain.groupId)

	if nodectx.GetNodeCtx().Node.Nodeopt.EnableSnapshot == false {
		chain_log.Debugf("<%s> Snapshot has been disabled on this node, skip", chain.groupId)
		return nil
	}

	if snapshot.SenderPubkey == chain.group.Item.OwnerPubKey {
		if chain.Consensus.SnapshotReceiver() != nil {
			//check signature
			verified, err := chain.Consensus.SnapshotReceiver().VerifySignature(snapshot)
			if err != nil {
				chain_log.Debugf("<%s> verify snapshot failed", chain.groupId)
				return err
			}

			if !verified {
				chain_log.Debugf("<%s> Invalid snapshot, signature invalid", chain.groupId)
				return errors.New("Invalid signature")
			}
			chain.Consensus.SnapshotReceiver().ApplySnapshot(snapshot)
		}
	} else {
		chain_log.Warningf("<%s> Snapshot from unknown source(not owner), pubkey <%s>", chain.groupId, snapshot.SenderPubkey)
	}

	return nil
}

func (chain *Chain) HandleBlockPsConn(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> HandleBlock called", chain.groupId)

	var shouldAccept bool

	bpk, err := localcrypto.Libp2pPubkeyToEthBase64(block.ProducerPubKey)
	if err != nil {
		bpk = block.ProducerPubKey
	}

	if chain.Consensus.Producer() != nil {
		//if I am a producer, no need to addBlock since block just produced is already saved
		chain_log.Debugf("<%s> Producer ignore incoming block", chain.groupId)
		shouldAccept = false
	} else if _, ok := chain.ProducerPool[bpk]; ok {
		//from registed producer
		chain_log.Debugf("<%s> User prepare to accept the block", chain.groupId)
		shouldAccept = true
	} else {
		//from someone else
		shouldAccept = false
		chain_log.Warningf("<%s> received block <%s> from unregisted producer <%s>, reject it", chain.group.Item.GroupId, block.BlockId, bpk)
	}

	if shouldAccept {
		err := chain.Consensus.User().AddBlock(block)
		if err != nil {
			chain_log.Debugf("<%s> user add block error <%s>", chain.groupId, err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				chain_log.Infof("<%s>, parent not exist, try sync backward from block <%s>", chain.groupId, block.BlockId)
				if chain.syncerrunner.Status == IDLE {
					return chain.syncerrunner.StartBackward(block.BlockId)
				} else {
					chain_log.Infof("<%s>, syncerrunner status is <%s>, so keep the new block, dont' sync backward", chain.groupId, chain.syncerrunner.Status)
				}
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

func (chain *Chain) handleReqBlockForward(trx *quorumpb.Trx, networktype conn.P2pNetworkType, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlockForward called", chain.groupId)
	if networktype == conn.PubSub {
		if chain.Consensus == nil || chain.Consensus.Producer() == nil {
			return nil
		}
		chain_log.Debugf("<%s> producer handleReqBlockForward called", chain.groupId)
		clientSyncerChannelId := conn.SYNC_CHANNEL_PREFIX + trx.GroupId + "_" + trx.SenderPubkey

		requester, blocks, isEmpty, err := chain.chaindata.GetBlockForward(trx)
		if err != nil {
			return err
		}

		//no block found
		if isEmpty {
			chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", chain.groupId)
			trx, err := chain.trxFactory.GetReqBlockRespTrx("", requester, blocks[0], quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
			if err != nil {
				return err
			}

			if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
				return err
			} else {
				return cmgr.SendTrxPubsub(trx, conn.SyncerChannel, clientSyncerChannelId)
			}
		}

		//send requested blocks out
		for _, block := range blocks {
			chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", chain.groupId)
			trx, err := chain.trxFactory.GetReqBlockRespTrx("", requester, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
			if err != nil {
				return err
			}
			if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
				return err
			} else {
				return cmgr.SendTrxPubsub(trx, conn.SyncerChannel, clientSyncerChannelId)
			}
		}
	} else if networktype == conn.RumExchange {
		subBlocks, err := chain.chaindata.GetBlockForwardByReqTrx(trx, chain.group.Item.CipherKey, chain.nodename)
		if err == nil {
			if len(subBlocks) > 0 {
				ks := nodectx.GetNodeCtx().Keystore
				mypubkey, err := ks.GetEncodedPubkey(chain.group.Item.GroupId, localcrypto.Sign)
				if err != nil {
					return err
				}
				for _, block := range subBlocks {
					reqBlockRespItem, err := chain.chaindata.CreateReqBlockResp(chain.group.Item.CipherKey, trx, block, mypubkey, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
					chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX) With RumExchange", chain.groupId)
					if err != nil {
						return err
					}

					bItemBytes, err := proto.Marshal(reqBlockRespItem)
					if err != nil {
						return err
					}

					trx, err := chain.trxFactory.CreateTrxByEthKey(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes, "")
					if err != nil {
						return err
					}

					if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
						return err
					} else {
						//reply to the source net stream
						return cmgr.SendTrxRex(trx, s)
					}
				}
			} else {
				chain_log.Debugf("no more block for <%s>, send ontop message?", chain.groupId)
			}

		} else {
			chain_log.Debugf("GetBlockForwardByReqTrx err %s", err)
		}
	}
	return nil
}

func (chain *Chain) handleReqBlockBackward(trx *quorumpb.Trx, networktype conn.P2pNetworkType, s network.Stream) error {
	if networktype == conn.PubSub {
		if chain.Consensus == nil || chain.Consensus.Producer() == nil {
			return nil
		}

		chain_log.Debugf("<%s> producer handleReqBlockForward called", chain.groupId)
		clientSyncerChannelId := conn.SYNC_CHANNEL_PREFIX + trx.GroupId + "_" + trx.SenderPubkey

		requester, block, isEmpty, err := chain.chaindata.GetBlockBackward(trx)
		if err != nil {
			return err
		}

		//no block found
		if isEmpty {
			chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", chain.groupId)
			trx, err := chain.trxFactory.GetReqBlockRespTrx("", requester, block, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
			if err != nil {
				return err
			}

			if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
				return err
			} else {
				return cmgr.SendTrxPubsub(trx, conn.SyncerChannel, clientSyncerChannelId)
			}
		}

		//send requested blocks out
		chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", chain.groupId)
		trx, err := chain.trxFactory.GetReqBlockRespTrx("", requester, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)

		if err != nil {
			return err
		}

		if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
			return err
		} else {
			return cmgr.SendTrxPubsub(trx, conn.SyncerChannel, clientSyncerChannelId)
		}

	} else if networktype == conn.RumExchange {
		//TOFIX: backward by RumExchange
		chain_log.Debugf("<%s> Backward by Rumexchange is not implemented", chain.groupId)
		return nil
		//reqblockid, block, err := chain.chaindata.GetBlockBackwardByReqTrx(trx, chain.group.Item.CipherKey, chain.nodename)
		//TOFIX use gsyncer
		//_ = reqblockid
		//if err == nil && block != nil {
		//	ks := nodectx.GetNodeCtx().Keystore
		//	mypubkey, err := ks.GetEncodedPubkey(chain.group.Item.GroupId, localcrypto.Sign)
		//	if err != nil {
		//		return err
		//	}
		//	reqBlockRespItem, err := chain.chaindata.CreateReqBlockResp(chain.group.Item.CipherKey, trx, block, mypubkey, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
		//	chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX) With RumExchange", chain.groupId)
		//	if err != nil {
		//		return err
		//	}

		//	bItemBytes, err := proto.Marshal(reqBlockRespItem)
		//	if err != nil {
		//		return err
		//	}

		//	trx, err := chain.trxFactory.CreateTrxByEthKey(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes, "")
		//	if err != nil {
		//		return err
		//	}

		//	if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
		//		return err
		//	} else {
		//		//reply to the source net stream
		//		return cmgr.SendTrxRex(trx, s)
		//	}

		//} else {
		//	chain_log.Debugf("GetBlockBackwordByReqTrx err %s", err)
		//}
	}
	return nil
}

func (chain *Chain) AddBlockSynced(resp *quorumpb.ReqBlockResp, block *quorumpb.Block) error {
	var err error
	providerpkey, err := localcrypto.Libp2pPubkeyToEthBase64(resp.ProviderPubkey)
	if err != nil && providerpkey == "" {
		chain_log.Warnf("<%s> Pubkey err <%s>", chain.groupId, err)
		return err
	}
	signpkey, err := localcrypto.Libp2pPubkeyToEthBase64(chain.group.Item.UserSignPubkey)
	if err != nil && signpkey == "" {
		chain_log.Warnf("<%s> Pubkey err <%s>", chain.groupId, err)
		return err
	}

	_, producer := chain.group.ChainCtx.ProducerPool[signpkey]

	if chain.syncerrunner.Status == SYNCING_FORWARD {
		if producer {
			err = chain.group.ChainCtx.AddBlock(block)
			if err == nil {
				chain_log.Debugf("<%s> SYNCING_FORWARD, PRODUCER ADD BLOCK", chain.groupId)
			}
		} else {
			err = chain.group.ChainCtx.Consensus.User().AddBlock(block)
			if err == nil {
				chain_log.Debugf("<%s> SYNCING_FORWARD, USER ADD BLOCK", chain.groupId)
			}
			//if err != nil {
			//	chain_log.Infof(err.Error())
			//}
		}
		//chain_log.Debugf("<%s> SYNCING_FORWARD, CONTINUE", chain.groupId)
	} else { //sync backward
		if producer {
			err = chain.group.ChainCtx.AddBlock(block)
			if err == nil {
				chain_log.Debugf("<%s> SYNCING_BACKWARD, PRODUCER ADD BLOCK", chain.groupId)
			}
		} else {
			err = chain.group.ChainCtx.Consensus.User().AddBlock(block)
			if err == nil {
				chain_log.Debugf("<%s> SYNCING_BACKWARD, USER ADD BLOCK", chain.groupId)
			}
		}

		if err == nil { //BACKWARD succ and block has been added, so switch to the FORWARD sync
			chain_log.Debugf("<%s> SYNCING_BACKWARD ADD BLOCK SUCC, CHANGE TO FORWARD", chain.groupId)
			chain.syncerrunner.SwapSyncDirection()

		}
		//chain_log.Debugf("<%s> SYNCING_BACKWARD, CONTINUE", chain.groupId)
	}
	return err
}

func (chain *Chain) HandleReqBlockResp(trx *quorumpb.Trx, waittask *BlockSyncTask) (string, error) {
	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
	if err != nil {
		return "", err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return "", err
	}

	reqBlockResp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(decryptData, reqBlockResp); err != nil {
		return "", err
	}

	//if not asked by myself, ignore it
	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		return "", errors.New("not ask by myself")
	}

	if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_NOT_FOUND { //sync done, set to IDLE
		chain_log.Debugf("<%s> receive BLOCK_NOT_FOUND response", chain.groupId)
		return reqBlockResp.BlockId, ErrSyncDone
	}
	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupId)

	newBlock := &quorumpb.Block{}

	if err := proto.Unmarshal(reqBlockResp.Block, newBlock); err != nil {
		return "", err
	}

	//check if the current wait task
	if waittask.Direction == Next && newBlock.PrevBlockId != waittask.BlockId { //err
		return "", errors.New("ignore this block")
	}

	var shouldAccept bool

	nbpk, err := localcrypto.Libp2pPubkeyToEthBase64(newBlock.ProducerPubKey)
	if err != nil {
		nbpk = newBlock.ProducerPubKey
	}

	chain_log.Debugf("<%s> REQ_BLOCK_RESP, block_id <%s>, block_producer <%s>", chain.groupId, newBlock.BlockId, nbpk)

	if _, ok := chain.ProducerPool[nbpk]; ok {
		shouldAccept = true
	} else {
		shouldAccept = false
	}

	if !shouldAccept {
		chain_log.Warnf(" <%s> Block producer <%s> not registed, reject", chain.groupId, nbpk)
		for key, _ := range chain.ProducerPool {
			chain_log.Warnf(" <%s> List Block producer %s", chain.groupId, key)
		}
		return "", errors.New("Block producer not registed")
	}

	return newBlock.BlockId, chain.AddBlockSynced(reqBlockResp, newBlock)
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
	producers, _ := nodectx.GetNodeCtx().GetChainStorage().GetProducers(chain.group.Item.GroupId, chain.nodename)
	for _, item := range producers {
		base64ethpkey, err := localcrypto.Libp2pPubkeyToEthBase64(item.ProducerPubkey)
		if err == nil {
			chain.ProducerPool[base64ethpkey] = item
		} else {
			chain.ProducerPool[item.ProducerPubkey] = item
		}
		ownerPrefix := "(producer)"
		if item.ProducerPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> Load producer <%s%s>", chain.groupId, item.ProducerPubkey, ownerPrefix)
	}

	connMgr, _ := conn.GetConn().GetConnMgr(chain.groupId)

	var producerspubkey []string
	for key, _ := range chain.ProducerPool {
		producerspubkey = append(producerspubkey, key)
	}

	connMgr.UpdProducers(producerspubkey)

	//update announced producer result
	announcedProducers, _ := nodectx.GetNodeCtx().GetChainStorage().GetAnnounceProducersByGroup(chain.group.Item.GroupId, chain.nodename)
	for _, item := range announcedProducers {
		_, ok := chain.ProducerPool[item.SignPubkey]
		err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounceResult(quorumpb.AnnounceType_AS_PRODUCER, chain.group.Item.GroupId, item.SignPubkey, ok, chain.nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.groupId, err.Error())
		}
	}
}

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
	users, _ := nodectx.GetNodeCtx().GetChainStorage().GetUsers(chain.group.Item.GroupId, chain.nodename)
	for _, item := range users {
		chain.userPool[item.UserPubkey] = item
		ownerPrefix := "(user)"
		if item.UserPubkey == chain.group.Item.OwnerPubKey {
			ownerPrefix = "(owner)"
		}
		chain_log.Infof("<%s> Load Users <%s_%s>", chain.groupId, item.UserPubkey, ownerPrefix)
	}

	//update announced User result
	announcedUsers, _ := nodectx.GetNodeCtx().GetChainStorage().GetAnnounceUsersByGroup(chain.group.Item.GroupId, chain.nodename)
	for _, item := range announcedUsers {
		_, ok := chain.userPool[item.SignPubkey]
		err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounceResult(quorumpb.AnnounceType_AS_USER, chain.group.Item.GroupId, item.SignPubkey, ok, chain.nodename)
		if err != nil {
			chain_log.Warningf("<%s> UpdAnnounceResult failed with error <%s>", chain.groupId, err.Error())
		}
	}
}

func (chain *Chain) GetSnapshotTag() (tag *quorumpb.SnapShotTag, err error) {
	if chain.Consensus.SnapshotReceiver() != nil {
		return chain.Consensus.SnapshotReceiver().GetTag(), nil
	} else {
		return nil, errors.New("Sender don't have snapshot tag")
	}
}

func (chain *Chain) CreateConsensus() error {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupId)

	var user def.User
	var producer def.Producer
	var snapshotreceiver chaindef.SnapshotReceiver
	var snapshotsender chaindef.SnapshotSender

	if chain.Consensus == nil || chain.Consensus.User() == nil {
		chain_log.Infof("<%s> Create and initial molasses user", chain.groupId)
		user = &consensus.MolassesUser{}
		user.Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
	} else {
		chain_log.Infof("<%s> reuse molasses user", chain.groupId)
		user = chain.Consensus.User()
	}

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(chain.group.Item.UserSignPubkey)
	if pk == "" {
		pk = chain.group.Item.UserSignPubkey
	}

	if _, ok := chain.ProducerPool[pk]; ok {
		if chain.Consensus == nil || chain.Consensus.Producer() == nil {
			chain_log.Infof("<%s> Create and initial molasses producer", chain.groupId)
			producer = &consensus.MolassesProducer{}
			producer.Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
		} else {
			chain_log.Infof("<%s> reuse molasses producer", chain.groupId)
			producer = chain.Consensus.Producer()
		}
	} else {
		chain_log.Infof("<%s> no producer created", chain.groupId)
		producer = nil
	}

	ownerpk, _ := localcrypto.Libp2pPubkeyToEthBase64(chain.group.Item.OwnerPubKey)
	if ownerpk == "" {
		ownerpk = chain.group.Item.OwnerPubKey
	}

	if ownerpk == pk {
		if chain.Consensus == nil || chain.Consensus.SnapshotSender() == nil {
			chain_log.Infof("<%s> Create and initial molasses SnapshotSender", chain.groupId)
			snapshotsender = &MolassesSnapshotSender{}
			snapshotsender.Init(chain.group.Item, chain.group.ChainCtx.nodename)
		} else {
			chain_log.Infof("<%s> reuse molasses snapshotsender", chain.groupId)
			snapshotsender = chain.Consensus.SnapshotSender()
		}
		snapshotreceiver = nil
	} else {
		if chain.Consensus == nil || chain.Consensus.SnapshotSender() == nil {
			chain_log.Infof("<%s> Create and initial molasses SnapshotReceiver", chain.groupId)
			snapshotreceiver = &MolassesSnapshotReceiver{}
			snapshotreceiver.Init(chain.group.Item, chain.group.ChainCtx.nodename)
		} else {
			chain_log.Infof("<%s> reuse molasses snapshot", chain.groupId)
			snapshotreceiver = chain.Consensus.SnapshotReceiver()
		}
		snapshotsender = nil
	}

	if chain.Consensus == nil {
		chain_log.Infof("<%s> created consensus", chain.groupId)
		chain.Consensus = consensus.NewMolasses(producer, user, snapshotsender, snapshotreceiver)
	} else {
		chain_log.Infof("<%s> reuse consensus", chain.groupId)
		chain.Consensus.SetProducer(producer)
		chain.Consensus.SetUser(user)
		chain.Consensus.SetSnapshotSender(snapshotsender)
		chain.Consensus.SetSnapshotReceiver(snapshotreceiver)
	}

	return nil
}

func (chain *Chain) TrxEnqueue(groupId string, trx *quorumpb.Trx) error {
	return TrxEnqueue(groupId, trx)
}

//func (chain *Chain) SyncForward(blockId string, nodename string) error {
//	chain_log.Debugf("<%s> SyncForward called", chain.groupId)
//	go func() {
//		//before start sync from other node, gather all local block and re-apply all trxs
//		chain_log.Debugf("<%s> Try find and chain all local blocks", chain.groupId)
//		chain_log.Debugf("<%s> height <%d>", chain.groupId, chain.group.Item.HighestHeight)
//		chain_log.Debugf("<%s> block_id <%s>", chain.groupId, chain.group.Item.HighestBlockId)
//
//		chain.syncer.SyncLocalBlock(blockId, nodename)
//		topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(chain.group.Item.HighestBlockId, false, nodename)
//		if err != nil {
//			chain_log.Warningf("Get top block error, blockId <%s>, <%s>", blockId, err.Error())
//			return
//		}
//		if chain.syncer != nil {
//			chain.syncer.SyncForward(topBlock)
//		}
//	}()
//
//	return nil
//}

//func (chain *Chain) SyncBackward(blockId string, nodename string) error {
//	chain_log.Debugf("<%s> SyncBackward called", chain.groupId)
//	go func() {
//		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(blockId, false, nodename)
//		if err != nil {
//			chain_log.Warningf("Get block error, blockId <%s>, <%s>", blockId, err.Error())
//			return
//		}
//
//		if chain.syncer != nil {
//			chain.syncer.SyncBackward(block)
//		}
//	}()
//
//	return nil
//}

func (chain *Chain) StartSync() error {
	chain_log.Debugf("<%s> StartSync called.", chain.groupId)

	if chain.group.Item.OwnerPubKey == chain.group.Item.UserSignPubkey {
		if len(chain.ProducerPool) == 1 {
			chain_log.Debugf("<%s> group owner, no registed producer, no need to sync", chain.group.Item.GroupId)
			return nil
		} else {
			chain_log.Debugf("<%s> owner, has registed producer, start sync missing block", chain.group.Item.GroupId)
		}
	} else if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
		chain_log.Debugf("<%s> producer, no need to sync forward (sync backward when new block produced and found missing block(s)", chain.group.Item.GroupId)
		return nil
	}
	chain.syncerrunner.Start(chain.group.Item.HighestBlockId)
	return nil
}
func (chain *Chain) StopSync() {
	chain_log.Debugf("<%s> StopSync called", chain.groupId)
	if chain.syncerrunner != nil {
		chain.syncerrunner.Stop()
	}
}

func (chain *Chain) IsSyncerIdle() bool {
	chain_log.Debugf("IsSyncerIdle called, groupId <%s>", chain.groupId)
	if chain.syncerrunner.Status == SYNCING_BACKWARD ||
		chain.syncerrunner.Status == SYNCING_FORWARD ||
		chain.syncerrunner.Status == LOCAL_SYNCING ||
		chain.syncerrunner.Status == SYNC_FAILED {
		chain_log.Debugf("<%s> syncer is busy, status: <%d>", chain.groupId, chain.syncerrunner.Status)
		return true
	}
	chain_log.Debugf("<%s> syncer is IDLE", chain.groupId)
	return false
}

func (chain *Chain) StartSnapshot() {
	chain_log.Debugf("<%s> StartSnapshot called", chain.groupId)
	if chain.group.Item.OwnerPubKey == chain.group.Item.UserSignPubkey {
		//I am producer, start snapshot ticker
		chain_log.Debugf("<%s> Owner start snapshot", chain.groupId)
		if chain.Consensus.SnapshotSender() == nil {
			chain_log.Debugf("<%s> snapshotsender is nil", chain.groupId)
			return
		}
		chain.Consensus.SnapshotSender().Start()
	}
}

func (chain *Chain) StopSnapshot() {
	chain_log.Debugf("<%s> StopSnapshot called", chain.groupId)
	if chain.group.Item.OwnerPubKey == chain.group.Item.UserSignPubkey {
		//I am producer, start snapshot ticker
		chain_log.Debugf("<%s> Owner stop snapshot", chain.groupId)
		if chain.Consensus.SnapshotSender() == nil {
			chain_log.Debugf("<%s> snapshotsender is nil", chain.groupId)
			return
		}
		chain.Consensus.SnapshotSender().Stop()
	}
}

func (chain *Chain) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	n, err := nodectx.GetDbMgr().GetNextNouce(groupId, nodeprefix)
	return n, err
}

func (chain *Chain) ApplyUserTrxs(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> applyTrxs called", chain.groupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			chain_log.Debugf("<%s> %s", chain.groupId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> existed, update trx only", chain.groupId, trx.TrxId)
			nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
			continue
		}

		originalData := trx.Data

		//new trx, apply it
		if trx.Type == quorumpb.TrxType_POST && chain.group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(chain.groupId, trx.Data)
			if err != nil {
				trx.Data = []byte("")
				//return err
			} else {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}

		} else {
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
		}

		//apply trx
		chain_log.Debugf("<%s> try apply trx <%s>", chain.groupId, trx.TrxId)

		//check if snapshotTag is available
		if trx.Type != quorumpb.TrxType_POST {
			snapshotTag, err := nodectx.GetNodeCtx().GetChainStorage().GetSnapshotTag(trx.GroupId, nodename)
			if err == nil && snapshotTag != nil {
				if snapshotTag.HighestHeight > chain.group.Item.HighestHeight {
					chain_log.Debugf("<%s> snapshotTag exist, trx already applied, ignore <%s>", chain.groupId, trx.TrxId)
					continue
				}
			}
		}

		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Debugf("<%s> apply POST trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().AddPost(trx, nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.UpdProducerList()
			chain.CreateConsensus()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.UpdUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_APP_CONFIG:
			chain_log.Debugf("<%s> apply APP_CONFIG trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAppConfigTrx(trx, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.groupId)
			err := nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
			if err != nil {
				chain_log.Errorf("<%s> handle CHAIN_CONFIG trx", chain.groupId)
			}
		case quorumpb.TrxType_SCHEMA:
			chain_log.Debugf("<%s> apply SCHEMA trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateSchema(trx, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.groupId, trx.Type)
		}

		//set trx data to original(encrypted)
		trx.Data = originalData

		//save trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}
	return nil
}

func (chain *Chain) ApplyProducerTrxs(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> applyTrxs called", chain.groupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			chain_log.Debugf("<%s> %s", chain.groupId, err.Error())
			continue
		}

		if isExist {
			chain_log.Debugf("<%s> trx <%s> existed, update trx", chain.groupId, trx.TrxId)
			nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
			continue
		}

		originalData := trx.Data

		if trx.Type == quorumpb.TrxType_POST && chain.group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			//just try decrypt it, if failed, save the original encrypted data
			//the reason for that is, for private group, before owner add producer, owner is the only producer,
			//since owner also needs to show POST data, and all announced user will encrypt for owner pubkey
			//owner can actually decrypt POST
			//for other producer, they can not decrpyt POST
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(chain.group.Item.GroupId, trx.Data)
			if err == nil {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}
		} else {
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
		}

		chain_log.Debugf("<%s> apply trx <%s>", chain.groupId, trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Debugf("<%s> apply POST trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().AddPost(trx, nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.UpdProducerList()
			chain.CreateConsensus()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.UpdUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_APP_CONFIG:
			chain_log.Debugf("<%s> apply APP_CONFIG trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAppConfigTrx(trx, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.groupId)
			err := nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
			if err != nil {
				chain_log.Errorf("<%s> handle CHAIN_CONFIG trx", chain.groupId)
			}
		case quorumpb.TrxType_SCHEMA:
			chain_log.Debugf("<%s> apply SCHEMA trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateSchema(trx, nodename)
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.groupId, trx.Type)
		}

		//set trx data to original (encrypted)
		trx.Data = originalData

		//save trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}

	return nil
}

//addBlock for producer
func (chain *Chain) AddBlock(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> AddBlock called", chain.groupId)

	//check if block is in storage
	isSaved, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.BlockId, false, chain.nodename)
	if err != nil {
		return err
	}
	if isSaved {
		return errors.New("Block already saved, ignore")
	}

	//check if block is in cache
	isCached, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(block.BlockId, true, chain.nodename)
	if err != nil {
		return err
	}

	if isCached {
		chain_log.Debugf("<%s> Block cached, update block", chain.groupId)
	}

	//Save block to cache
	err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, true, chain.nodename)
	if err != nil {
		return err
	}

	parentExist, err := nodectx.GetNodeCtx().GetChainStorage().IsParentExist(block.PrevBlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		chain_log.Debugf("<%s> parent of block <%s> is not exist", chain.groupId, block.BlockId)
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(block.PrevBlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := rumchaindata.IsBlockValid(block, parentBlock)
	if !valid {
		chain_log.Debugf("<%s> remove invalid block <%s> from cache", chain.groupId, block.BlockId)
		chain_log.Warningf("<%s> invalid block <%s>", chain.groupId, err.Error())
		return nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.BlockId, true, chain.nodename)
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetNodeCtx().GetChainStorage().GatherBlocksFromCache(block, true, chain.nodename)
	if err != nil {
		return err
	}

	//get all trxs in those new blocks
	var trxs []*quorumpb.Trx
	trxs, err = rumchaindata.GetAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply those trxs
	err = chain.ApplyProducerTrxs(trxs, chain.nodename)
	if err != nil {
		return err
	}

	//move blocks from cache to normal
	for _, block := range blocks {
		chain_log.Debugf("<%s> move block <%s> from cache to chain", chain.groupId, block.BlockId)
		err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(block, false, chain.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetNodeCtx().GetChainStorage().RmBlock(block.BlockId, true, chain.nodename)
		if err != nil {
			return err
		}
	}

	for _, block := range blocks {
		err := nodectx.GetNodeCtx().GetChainStorage().AddProducedBlockCount(chain.groupId, block.ProducerPubKey, chain.nodename)
		if err != nil {
			return err
		}
	}

	chain_log.Debugf("<%s> chain height before recal: <%d>", chain.groupId, chain.group.Item.HighestHeight)
	topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(chain.group.Item.HighestBlockId, false, chain.nodename)
	if err != nil {
		return err
	}
	newHeight, newHighestBlockId, err := chain.RecalChainHeight(blocks, chain.group.Item.HighestHeight, topBlock, chain.nodename)
	if err != nil {
		return err
	}
	chain_log.Debugf("<%s> new height <%d>, new highest blockId %v", chain.groupId, newHeight, newHighestBlockId)

	return chain.UpdChainInfo(newHeight, newHighestBlockId)
}

func (chain *Chain) GetSyncStatus() int8 {
	return chain.syncerrunner.Status
}

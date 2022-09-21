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
	Consensus          def.Consensus
	ProviderPeerIdPool map[string]string
	trxFactory         *rumchaindata.TrxFactory
	syncerrunner       *SyncerRunner

	chaindata *ChainData
}

func (chain *Chain) NewChain(group *Group) error {
	chain_log.Debugf("<%s> NewChain called", group.Item.GroupId)
	chain.group = group

	chain.nodename = nodectx.GetNodeCtx().Name
	chain.groupId = group.Item.GroupId

	chain.trxFactory = &rumchaindata.TrxFactory{}
	chain.trxFactory.Init(nodectx.GetNodeCtx().Version, group.Item, chain.nodename, chain)

	chain.syncerrunner = NewSyncerRunner(group, chain, chain.nodename)
	chain.chaindata = &ChainData{nodename: chain.nodename, groupId: group.Item.GroupId, groupCipherKey: group.Item.CipherKey, userSignPubkey: group.Item.UserSignPubkey, dbmgr: nodectx.GetDbMgr()}
	return nil
}

func (chain *Chain) GetNodeName() string {
	return chain.nodename
}

func (chain *Chain) SetRumExchangeTestMode() {
	chain_log.Debugf("<%s> SetRumExchangeTestMode called", chain.groupId)
	//chain.syncer.SetRumExchangeTestMode()
}

func (chain *Chain) GetChainSyncIface() chaindef.ChainDataSyncIface {
	chain_log.Debugf("<%s> GetChainSyncIface called", chain.groupId)
	return chain
}

func (chain *Chain) GetTrxFactory() chaindef.TrxFactoryIface {
	chain_log.Debugf("<%s> GetTrxFactory called", chain.groupId)
	return chain.trxFactory
}

/*
func (chain *Chain) GetSyncer() *Syncer {
	return chain.syncer
}
*/

func (chain *Chain) GetPubqueueIface() chaindef.PublishQueueIface {
	chain_log.Debugf("<%s> GetPubqueueIface called", chain.groupId)
	return GetPubQueueWatcher()
}

func (chain *Chain) UpdChainInfo(Epoch int64) error {
	chain_log.Debugf("<%s> UpdChainInfo called", chain.groupId)
	chain.group.Item.Epoch = Epoch
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	chain_log.Infof("<%s> Chain Info updated, latest Epoch %d", chain.group.Item.GroupId, Epoch)
	return nodectx.GetNodeCtx().GetChainStorage().UpdGroup(chain.group.Item)
}

/*
	PSConn handler
*/

//Handle Trx from PsConn
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

	switch trx.Type {
	case quorumpb.TrxType_POST,
		quorumpb.TrxType_ANNOUNCE,
		quorumpb.TrxType_PRODUCER,
		quorumpb.TrxType_USER,
		quorumpb.TrxType_SCHEMA,
		quorumpb.TrxType_APP_CONFIG,
		quorumpb.TrxType_CHAIN_CONFIG:
		chain.producerAddTrx(trx)
		break
	case quorumpb.TrxType_REQ_BLOCK_FORWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockForward(trx, conn.PubSub, nil)
	//backward sync removed
	//case quorumpb.TrxType_REQ_BLOCK_BACKWARD:
	//	if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
	//		return nil
	//	}
	//	chain.handleReqBlockBackward(trx, conn.PubSub, nil)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.syncerrunner.AddTrxToSyncerQueue(trx)
		//err := chain.handleReqBlockResp(trx)
		//return err
	default:
		chain_log.Warningf("<%s> unsupported msg type", chain.group.Item.GroupId)
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

//handle BLOCK msg from PSconn
func (chain *Chain) HandleBlockPsConn(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> HandleBlock called", chain.groupId)

	bpk, err := localcrypto.Libp2pPubkeyToEthBase64(block.BookkeepingPubkey)
	if err != nil {
		bpk = block.BookkeepingPubkey
	}

	//from registed producer
	if _, ok := chain.ProducerPool[bpk]; !ok {
		chain_log.Warningf("<%s> received block <%s> from unregisted producer <%s>, reject it", chain.group.Item.GroupId, block.Epoch, bpk)
		return nil
	} else {
		if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
			//I am a producer but not in promoted producer list
			if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; !ok {
				err := chain.Consensus.Producer().AddBlock(block)
				if err != nil {
					chain_log.Debugf("<%s> producer add block error <%s>", chain.groupId, err.Error())
					if err.Error() == "PARENT_NOT_EXIST" {
						chain_log.Infof("<%s>, parent not exist, sync backward from block <%d>", chain.groupId, block.Epoch)
						//return chain.syncer.SyncBackward(block)
					}
				}
			}
		} else {
			err := chain.Consensus.User().AddBlock(block)
			if err != nil {
				chain_log.Debugf("<%s> user add block error <%s>", chain.groupId, err.Error())
				if err.Error() == "PARENT_NOT_EXIST" {
					chain_log.Infof("<%s>, parent not exist, sync backward from block <%d>", chain.groupId, block.Epoch)
					//return chain.syncer.SyncBackward(block)
				}
			}
		}
	}

	return nil
}

func (chain *Chain) HandleHBPsConn(hb *quorumpb.HBMsg) error {

	//non producer node should not handle hb msg
	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; !ok {
		return nil
	}

	if chain.Consensus.Producer() == nil {
		return nil
	}

	return chain.Consensus.Producer().HandleHBMsg(hb)
}

/*
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
*/

/*
	Rex Handler
*/

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
		chain.HandleReqBlockResp(trx)
	default:
		//do nothing
	}

	return nil
}

func (chain *Chain) HandleBlockRex(block *quorumpb.Block, s network.Stream) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.groupId)
	return nil
}

func (chain *Chain) HandleHBRex(hb *quorumpb.HBMsg) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.groupId)
	return nil
}

func (chain *Chain) producerAddTrx(trx *quorumpb.Trx) error {
	if chain.Consensus != nil && chain.Consensus.Producer() == nil {
		return nil
	}

	//not in group producer list, do nothing
	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; !ok {
		return nil
	}

	chain_log.Debugf("<%s> producerAddTrx called", chain.groupId)
	chain.Consensus.Producer().AddTrx(trx)
	return nil
}

func (chain *Chain) handleReqBlockForward(trx *quorumpb.Trx, networktype conn.P2pNetworkType, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlockForward called", chain.groupId)

	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return nil
	}

	//TODO: check my sync status, only response when the status is IDLE
	//if chain.GetSyncerStatus() != IDLE {
	//	return nil
	//}
	chain_log.Debugf("<%s> producer handleReqBlockForward called", chain.groupId)
	clientSyncerChannelId := conn.SYNC_CHANNEL_PREFIX + trx.GroupId + "_" + trx.SenderPubkey
	requester, block, isEmpty, err := chain.chaindata.GetBlockForward(trx)
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
	//for _, block := range blocks {
	chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", chain.groupId)
	blockresptrx, err := chain.trxFactory.GetReqBlockRespTrx("", requester, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
	if err != nil {
		return err
	}
	if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
		return err
	} else {
		return cmgr.SendTrxPubsub(blockresptrx, conn.SyncerChannel, clientSyncerChannelId)
	}
	//}
	/*
		if networktype == conn.PubSub {
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

	*/
	return nil
}

func (chain *Chain) handleReqBlockBackward(trx *quorumpb.Trx, networktype conn.P2pNetworkType, s network.Stream) error {
	chain_log.Debugf("<%s> handleReqBlockBackward called", chain.groupId)
	/*
		if networktype == conn.PubSub {
			if chain.Consensus == nil || chain.Consensus.Producer() == nil {
				return nil
			}

			chain_log.Debugf("<%s> producer handleReqBlockBackward called", chain.groupId)
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
			block, err := chain.chaindata.GetBlockBackwardByReqTrx(trx, chain.group.Item.CipherKey, chain.nodename)
			if err == nil && block != nil {
				ks := nodectx.GetNodeCtx().Keystore
				mypubkey, err := ks.GetEncodedPubkey(chain.group.Item.GroupId, localcrypto.Sign)
				if err != nil {
					return err
				}
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

			} else {
				chain_log.Debugf("GetBlockBackwordByReqTrx err %s", err)
			}
		}

	*/
	return nil
}

func (chain *Chain) HandleReqBlockResp(trx *quorumpb.Trx) (int64, error) {
	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupId)
	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
	if err != nil {
		return 0, err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return 0, err
	}

	reqBlockResp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(decryptData, reqBlockResp); err != nil {
		return 0, err
	}
	//TODO: Verify response and block
	chain_log.Debugf("<%s> ======TODO: handleReqBlockResp Verify response and block ", chain.groupId)

	if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_NOT_FOUND { //sync done, set to IDLE
		chain_log.Debugf("<%s> receive BLOCK_NOT_FOUND response", chain.groupId)
		return reqBlockResp.Epoch, ErrSyncDone
	}

	//if not asked by myself, ignore it
	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		return 0, nil
	}

	newBlock := &quorumpb.Block{}

	if err := proto.Unmarshal(reqBlockResp.Block, newBlock); err != nil {
		return 0, err
	}

	chain_log.Debugf("<%s> newBlock.Epoch is %d  waitepoch is %d .", chain.groupId, newBlock.Epoch, chain.syncerrunner.GetWaitEpoch())
	if newBlock.Epoch != chain.syncerrunner.GetWaitEpoch() {
		chain_log.Debugf("<%s> Ingore newBlock, return", chain.groupId)
		return 0, nil

	}

	//TODO: if block epoch < waiting epoch, ignore it.
	var shouldAccept bool
	shouldAccept = true
	//TODO: verify the block
	//if run as producer node
	chain_log.Debugf("<%s> ======TODO: handleReqBlockResp set shouldAccept", chain.groupId)

	if !shouldAccept {
		chain_log.Debugf("The block can't be accepted, reason:...")
		return 0, nil
	}
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		chain_log.Info("PRODUCER_NODE handle block")
		return newBlock.Epoch + 1, chain.Consensus.Producer().AddBlock(newBlock)
	} else {
		//user sync
		return newBlock.Epoch + 1, chain.Consensus.User().AddBlock(newBlock)
	}
}

func (chain *Chain) UpdProducerList() {
	chain_log.Debugf("<%s> UpdProducerList called", chain.groupId)
	//create and load group producer pool
	chain.ProducerPool = make(map[string]*quorumpb.ProducerItem)
	producers, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(chain.group.Item.GroupId, chain.nodename)

	if err != nil {
		chain_log.Infof("Get producer failed with err %s", err.Error())
	}

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
}

func (chain *Chain) UpdConnMgrProducer() {

	connMgr, _ := conn.GetConn().GetConnMgr(chain.groupId)

	var producerspubkey []string
	for key, _ := range chain.ProducerPool {
		producerspubkey = append(producerspubkey, key)
	}

	connMgr.UpdProducers(producerspubkey)
}

func (chain *Chain) UpdAnnouncedProducerStatus() {
	chain_log.Debugf("<%s> UpdAnnouncedProducerStatus called", chain.groupId)
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

func (chain *Chain) UpdProducerConfig() {
	chain_log.Debugf("<%s> UpdProducerConfig called", chain.groupId)
	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return
	}

	//recreate producer BFT config
	chain.Consensus.Producer().RecreateBft()
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
	/*
		if chain.Consensus.SnapshotReceiver() != nil {
			return chain.Consensus.SnapshotReceiver().GetTag(), nil
		} else {
			return nil, errors.New("Sender don't have snapshot tag")
		}
	*/

	return nil, nil
}

func (chain *Chain) CreateConsensus() error {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupId)

	var user def.User
	var producer def.Producer

	var shouldCreateUser, shouldCreateProducer bool

	//create user/producer when run as FULL_NODE
	//only create producer when run PRODUCER_NODE
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		shouldCreateProducer = true
		shouldCreateUser = false
	} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
		//check if I am owner of the group
		if chain.group.Item.UserSignPubkey == chain.group.Item.OwnerPubKey {
			shouldCreateProducer = true
		} else {
			shouldCreateProducer = false
		}
		shouldCreateUser = true
	} else {
		return errors.New("Unknow nodetype")
	}

	if shouldCreateProducer {
		//create producer anyway
		//if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		chain_log.Infof("<%s> Create and initial molasses producer", chain.groupId)
		producer = &consensus.MolassesProducer{}
		producer.NewProducer(chain.group.Item, chain.group.ChainCtx.nodename, chain)

		/*
			} else {
				chain_log.Infof("<%s> reuse molasses producer", chain.groupId)
				producer = chain.Consensus.Producer()
				producer.RecreateBft()
			}
		*/
	}

	if shouldCreateUser {

		//if chain.Consensus == nil || chain.Consensus.User() == nil {
		chain_log.Infof("<%s> Create and initial molasses user", chain.groupId)
		user = &consensus.MolassesUser{}
		user.NewUser(chain.group.Item, chain.group.ChainCtx.nodename, chain)
		/*
				} else {
				chain_log.Infof("<%s> reuse molasses user", chain.groupId)
				user = chain.Consensus.User()
			}
		*/
	}

	/*
		var snapshotreceiver chaindef.SnapshotReceiver
		var snapshotsender chaindef.SnapshotSender

		pk, _ := localcrypto.Libp2pPubkeyToEthBase64(chain.group.Item.UserSignPubkey)
		if pk == "" {
			pk = chain.group.Item.UserSignPubkey
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
	*/

	//if chain.Consensus == nil {
	chain_log.Infof("<%s> create new consensus", chain.groupId)
	chain.Consensus = consensus.NewMolasses(producer, user /*, snapshotsender, snapshotreceiver */)

	/*

		} else {
			chain_log.Infof("<%s> reuse consensus", chain.groupId)
			chain.Consensus.SetProducer(producer)
			chain.Consensus.SetUser(user)
			//chain.Consensus.SetSnapshotSender(snapshotsender)
			//chain.Consensus.SetSnapshotReceiver(snapshotreceiver)
		}
	*/

	return nil
}

func (chain *Chain) TrxEnqueue(groupId string, trx *quorumpb.Trx) error {
	return TrxEnqueue(groupId, trx)
}

func (chain *Chain) StartSync() error {
	chain_log.Debugf("<%s> StartSync called.", chain.groupId)
	//all producers and owner must do sync after service start.
	//if chain.group.Item.OwnerPubKey == chain.group.Item.UserSignPubkey {
	//	if len(chain.ProducerPool) == 1 {
	//		chain_log.Debugf("<%s> group owner, no registed producer, no need to sync", chain.group.Item.GroupId)
	//		return nil
	//	} else {
	//		chain_log.Debugf("<%s> owner, has registed producer, start sync missing block", chain.group.Item.GroupId)
	//	}
	//} else if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
	//	chain_log.Debugf("<%s> producer, no need to sync forward (sync backward when new block produced and found missing block(s)", chain.group.Item.GroupId)
	//	return nil
	//}
	chain_log.Debugf("<%s> StartSync from %d", chain.groupId, chain.group.Item.Epoch)
	chain.syncerrunner.Start(chain.group.Item.Epoch + 1)
	return nil
}

//func (chain *Chain) SyncForward(epoch int64, nodename string) error {
//	chain_log.Debugf("<%s> SyncForward called", chain.groupId)
//	go func() {
//		//before start sync from other node, gather all local block and re-apply all trxs
//		//chain_log.Debugf("<%s> Try find and chain all local blocks", chain.groupId)
//		//chain.syncer.SyncLocalBlock(epoch, nodename)
//		//topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(chain.group.Item.GroupId, chain.group.Item.Epoch, false, nodename)
//		//if err != nil {
//		//	chain_log.Warningf("Get top block error, epoch <%d>, <%s>", epoch, err.Error())
//		//	return
//		//}
//		//if chain.syncer != nil {
//		//	chain.syncer.SyncForward(topBlock)
//		//}
//	}()
//
//	return nil
//}

func (chain *Chain) StopSync() error {
	chain_log.Debugf("<%s> StopSync called", chain.groupId)
	chain_log.Debugf("<%s> ======TODO: cal syncerrunner to stop", chain.groupId)
	//before start sync from other node, gather all
	//if chain.syncer != nil {
	//	return chain.syncer.StopSync()
	//}
	return nil
}
func (chain *Chain) GetSyncerStatus() int8 {
	return chain.syncerrunner.Status
}

/*

func (chain *Chain) SyncBackward(epoch int64, nodename string) error {
	chain_log.Debugf("<%s> SyncBackward called", chain.groupId)
	go func() {
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(chain.group.Item.GroupId, epoch, false, nodename)
		if err != nil {
			chain_log.Warningf("Get block error, epoch <%s>, <%s>", epoch, err.Error())
			return
		}

		if chain.syncer != nil {
			chain.syncer.SyncBackward(block)
		}
	}()

	return nil
}


func (chain *Chain) IsSyncerIdle() bool {
	chain_log.Debugf("IsSyncerIdle called, groupId <%s>", chain.groupId)

	if chain.syncer.Status == SYNCING_BACKWARD ||
		chain.syncer.Status == SYNCING_FORWARD ||
		chain.syncer.Status == LOCAL_SYNCING ||
		chain.syncer.Status == SYNC_FAILED {
		chain_log.Debugf("<%s> syncer is busy, status: <%d>", chain.groupId, chain.syncer.Status)
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
*/

func (chain *Chain) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	n, err := nodectx.GetDbMgr().GetNextNouce(groupId, nodeprefix)
	return n, err
}

func (chain *Chain) ApplyTrxsFullNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsFullNode called", chain.groupId)
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

		/*
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
		*/
		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Debugf("<%s> apply POST trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().AddPost(trx, nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.UpdProducerList()
			chain.UpdAnnouncedProducerStatus()
			chain.UpdProducerConfig()
			//chain.UpdConnMgrProducer()
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

func (chain *Chain) ApplyTrxsProducerNode(trxs []*quorumpb.Trx, nodename string) error {
	chain_log.Debugf("<%s> ApplyTrxsProducerNode called", chain.groupId)
	for _, trx := range trxs {
		if trx.Type == quorumpb.TrxType_APP_CONFIG || trx.Type == quorumpb.TrxType_POST {
			//producer node does not handle APP_CONFIG and POST
			chain_log.Infof("Skip TRX %s with type %s", trx.TrxId, trx.Type.String())
			continue
		}

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

		chain_log.Debugf("<%s> apply trx <%s>", chain.groupId, trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_PRODUCER:
			chain_log.Debugf("<%s> apply PRODUCER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateProducerTrx(trx, nodename)
			chain.UpdProducerList()
			chain.UpdAnnouncedProducerStatus()
			chain.UpdProducerConfig()
			chain.UpdConnMgrProducer()
		case quorumpb.TrxType_USER:
			chain_log.Debugf("<%s> apply USER trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateUserTrx(trx, nodename)
			chain.UpdUserList()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Debugf("<%s> apply ANNOUNCE trx", chain.groupId)
			nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(trx.Data, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			chain_log.Debugf("<%s> apply CHAIN_CONFIG trx", chain.groupId)
			err := nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfigTrx(trx, nodename)
			if err != nil {
				chain_log.Errorf("<%s> handle CHAIN_CONFIG trx", chain.groupId)
			}
		default:
			chain_log.Warningf("<%s> unsupported msgType <%s>", chain.groupId, trx.Type)
		}

		trx.Data = originalData

		//save trx to db
		nodectx.GetNodeCtx().GetChainStorage().AddTrx(trx, nodename)
	}

	return nil
}

func (chain *Chain) AddSyncedBlock(block *quorumpb.Block) error {
	chain_log.Debugf("<%s> AddBlock called", chain.groupId)

	/*
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

	*/

	return nil
}

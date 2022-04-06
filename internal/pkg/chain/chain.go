package chain

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
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
	Consensus          Consensus
	producerChannTimer *time.Timer
	ProviderPeerIdPool map[string]string
	trxFactory         *TrxFactory

	syncer *Syncer
}

func (chain *Chain) Init(group *Group) error {
	chain_log.Debugf("<%s> Init called", group.Item.GroupId)
	chain.group = group

	chain.nodename = nodectx.GetNodeCtx().Name
	chain.groupId = group.Item.GroupId
	chain.chaindata = &ChainData{nodectx.GetDbMgr()}

	chain.trxFactory = &TrxFactory{}
	chain.trxFactory.Init(group.Item, chain.nodename)

	chain.syncer = &Syncer{}
	chain.syncer.Init(group, chain)

	chain_log.Infof("<%s> chainctx initialed", chain.groupId)
	return nil
}

func (chain *Chain) SetRumExchangeTestMode() {
	chain.syncer.SetRumExchangeTestMode()
}

func (chain *Chain) GetChainCtx() *Chain {
	return chain
}

func (chain *Chain) GetTrxFactory() *TrxFactory {
	return chain.trxFactory
}

func (chain *Chain) GetSyncer() *Syncer {
	return chain.syncer
}

func (chain *Chain) UpdChainInfo(height int64, blockId string) error {
	chain_log.Debugf("<%s> UpdChainInfo called", chain.groupId)
	chain.group.Item.HighestHeight = height
	chain.group.Item.HighestBlockId = blockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	chain_log.Infof("<%s> Chain Info updated %d, %v", chain.group.Item.GroupId, height, blockId)
	return nodectx.GetDbMgr().UpdGroup(chain.group.Item)
}

func (chain *Chain) HandleTrxRex(trx *quorumpb.Trx, from peer.ID) error {
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
		chain.handleReqBlockForward(trx, conn.RumExchange, from)
	case quorumpb.TrxType_REQ_BLOCK_BACKWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockBackward(trx, conn.RumExchange, from)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockResp(trx)
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

	verified, err := chain.trxFactory.VerifyTrx(trx)
	if err != nil {
		chain_log.Warnf("<%s> verify Trx failed with err <%s>", chain.groupId, err.Error())
		return errors.New("Verify Trx failed")
	}

	if !verified {
		chain_log.Warnf("<%s> Invalid Trx, signature verify failed, sender %s", chain.groupId, trx.SenderPubkey)
		return errors.New("Invalid Trx")
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
		chain.handleReqBlockForward(trx, conn.PubSub, "")
	case quorumpb.TrxType_REQ_BLOCK_BACKWARD:
		if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
			return nil
		}
		chain.handleReqBlockBackward(trx, conn.PubSub, "")
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

func (chain *Chain) HandleBlockRex(block *quorumpb.Block, from peer.ID) error {
	chain_log.Debugf("<%s> HandleBlockRex called", chain.groupId)
	return nil
}

func (chain *Chain) HandleSnapshotPsConn(snapshot *quorumpb.Snapshot) error {
	chain_log.Debugf("<%s> HandleSnapshotPsConn called", chain.groupId)
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
	if chain.Consensus.Producer() != nil {
		//if I am a producer, no need to addBlock since block just produced is already saved
		chain_log.Debugf("<%s> Producer ignore incoming block", chain.groupId)
		shouldAccept = false
	} else if _, ok := chain.ProducerPool[block.ProducerPubKey]; ok {
		//from registed producer
		chain_log.Debugf("<%s> User prepare to accept the block", chain.groupId)
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
				return chain.syncer.SyncBackward(block)
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

func (chain *Chain) handleReqBlockForward(trx *quorumpb.Trx, networktype conn.P2pNetworkType, from peer.ID) error {
	chain_log.Debugf("<%s> handleReqBlockForward called", chain.groupId)
	if networktype == conn.PubSub {
		if chain.Consensus == nil || chain.Consensus.Producer() == nil {
			return nil
		}
		chain_log.Debugf("<%s> producer handleReqBlockForward called", chain.groupId)
		clientSyncerChannelId := conn.SYNC_CHANNEL_PREFIX + trx.GroupId + "_" + trx.SenderPubkey

		requester, blocks, isEmpty, err := chain.Consensus.Producer().GetBlockForward(trx)
		if err != nil {
			return err
		}

		//no block found
		if isEmpty {
			chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", chain.groupId)
			trx, err := chain.trxFactory.GetReqBlockRespTrx(requester, blocks[0], quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
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
			trx, err := chain.trxFactory.GetReqBlockRespTrx(requester, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
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

					trx, err := chain.trxFactory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes)
					if err != nil {
						return err
					}

					if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
						return err
					} else {
						return cmgr.SendTrxRex(trx, from)
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

func (chain *Chain) handleReqBlockBackward(trx *quorumpb.Trx, networktype conn.P2pNetworkType, from peer.ID) error {
	if networktype == conn.PubSub {
		if chain.Consensus == nil || chain.Consensus.Producer() == nil {
			return nil
		}

		chain_log.Debugf("<%s> producer handleReqBlockForward called", chain.groupId)
		clientSyncerChannelId := conn.SYNC_CHANNEL_PREFIX + trx.GroupId + "_" + trx.SenderPubkey

		requester, block, isEmpty, err := chain.Consensus.Producer().GetBlockBackward(trx)
		if err != nil {
			return err
		}

		//no block found
		if isEmpty {
			chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", chain.groupId)
			trx, err := chain.trxFactory.GetReqBlockRespTrx(requester, block, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
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
		trx, err := chain.trxFactory.GetReqBlockRespTrx(requester, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)

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

			trx, err := chain.trxFactory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes)
			if err != nil {
				return err
			}

			var pkg *quorumpb.Package
			pkg = &quorumpb.Package{}
			pbBytes, err := proto.Marshal(trx)
			if err != nil {
				return err
			}
			pkg.Type = quorumpb.PackageType_TRX
			pkg.Data = pbBytes

			rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
			nodectx.GetNodeCtx().Node.RumExchange.PublishTo(rummsg, from)

		} else {
			chain_log.Debugf("GetBlockBackwordByReqTrx err %s", err)
		}
	}
	return nil
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

	reqBlockResp := &quorumpb.ReqBlockResp{}
	if err := proto.Unmarshal(decryptData, reqBlockResp); err != nil {
		return err
	}

	//if not asked by myself, ignore it
	if reqBlockResp.RequesterPubkey != chain.group.Item.UserSignPubkey {
		return nil
	}

	chain_log.Debugf("<%s> handleReqBlockResp called", chain.groupId)

	newBlock := &quorumpb.Block{}

	if err := proto.Unmarshal(reqBlockResp.Block, newBlock); err != nil {
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

	return chain.syncer.AddBlockSynced(reqBlockResp, newBlock)
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

	connMgr, _ := conn.GetConn().GetConnMgr(chain.groupId)

	var producerspubkey []string
	for key, _ := range chain.ProducerPool {
		producerspubkey = append(producerspubkey, key)
	}

	connMgr.UpdProducers(producerspubkey)

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

/*
func (chain *Chain) HandleAskPeerID(trx *quorumpb.Trx) error {
	chain_log.Debugf("<%s> HandleAskPeerID called", chain.groupId)
	if chain.Consensus == nil || chain.Consensus.Producer() == nil {
		return nil
	}

	var reqItem quorumpb.AskPeerId
	ciperKey, err := hex.DecodeString(chain.group.Item.CipherKey)
	if err != nil {
		return err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return err
	}

	if err := proto.Unmarshal(decryptData, &reqItem); err != nil {
		return err
	}

	//check if requester is in group block list
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey, chain.nodename)

	if isBlocked {
		chain_log.Debugf("<%s> user <%s> is blocked", chain.groupId, trx.SenderPubkey)
		return nil
	}

	var respItem *quorumpb.AskPeerIdResp
	respItem = &quorumpb.AskPeerIdResp{}

	respItem.GroupId = chain.groupId
	respItem.RespPeerId = nodectx.GetNodeCtx().PeerId.Pretty()
	respItem.RespPeerPubkey = chain.group.Item.UserSignPubkey

	trx, err = chain.trxFactory.GetAskPeerIdRespTrx(respItem)

	if cmgr, err := conn.GetConn().GetConnMgr(chain.groupId); err != nil {
		return err
	} else {
		return cmgr.SendTrxPubsub(trx, conn.ProducerChannel)
	}
}
*/

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

//func (chain *Chain) InitSession(channelId string) error {
//chain_log.Debugf("<%s> InitSession called", chain.groupId)
//if peerId, ok := chain.ProviderPeerIdPool[chain.group.Item.OwnerPubKey]; ok {
//	return nodectx.GetNodeCtx().Node.RumSession.InitSession(peerId, channelId)
//} else {
//	return chain.AskPeerId()
//}
// return nil
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
		chain_log.Infof("<%s> Load Users <%s_%s>", chain.groupId, item.UserPubkey, ownerPrefix)
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

func (chain *Chain) GetSnapshotTag() (tag *quorumpb.SnapShotTag, err error) {
	if chain.Consensus.SnapshotReceiver() != nil {
		return chain.Consensus.SnapshotReceiver().GetTag(), nil
	} else {
		return nil, errors.New("Sender don't have snapshot tag")
	}
}

func (chain *Chain) CreateConsensus() error {
	chain_log.Debugf("<%s> CreateConsensus called", chain.groupId)

	var user User
	var producer Producer
	var snapshotreceiver SnapshotReceiver
	var snapshotsender SnapshotSender

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
		} else {
			chain_log.Infof("<%s> reuse molasses producer", chain.groupId)
			producer = chain.Consensus.Producer()
		}
	} else {
		chain_log.Infof("<%s> no producer created", chain.groupId)
		producer = nil
	}

	if chain.group.Item.OwnerPubKey == chain.group.Item.UserSignPubkey {
		if chain.Consensus == nil || chain.Consensus.SnapshotSender() == nil {
			chain_log.Infof("<%s> Create and initial molasses SnapshotSender", chain.groupId)
			snapshotsender = &MolassesSnapshotSender{}
			snapshotsender.Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
		} else {
			chain_log.Infof("<%s> reuse molasses snapshotsender", chain.groupId)
			snapshotsender = chain.Consensus.SnapshotSender()
		}
		snapshotreceiver = nil
	} else {
		if chain.Consensus == nil || chain.Consensus.SnapshotSender() == nil {
			chain_log.Infof("<%s> Create and initial molasses SnapshotReceiver", chain.groupId)
			snapshotreceiver = &MolassesSnapshotReceiver{}
			snapshotreceiver.Init(chain.group.Item, chain.group.ChainCtx.nodename, chain)
		} else {
			chain_log.Infof("<%s> reuse molasses snapshot", chain.groupId)
			snapshotreceiver = chain.Consensus.SnapshotReceiver()
		}
		snapshotsender = nil
	}

	if chain.Consensus == nil {
		chain_log.Infof("<%s> created consensus", chain.groupId)
		chain.Consensus = NewMolasses(producer, user, snapshotsender, snapshotreceiver)
	} else {
		chain_log.Infof("<%s> reuse consensus", chain.groupId)
		chain.Consensus.SetProducer(producer)
		chain.Consensus.SetUser(user)
		chain.Consensus.SetSnapshotSender(snapshotsender)
		chain.Consensus.SetSnapshotReceiver(snapshotreceiver)
	}

	return nil
}

func (chain *Chain) SyncForward(blockId string, nodename string) error {
	chain_log.Debugf("<%s> SyncForward called", chain.groupId)
	go func() {
		//before start sync from other node, gather all local block and re-apply all trxs
		chain_log.Debugf("<%s> Try find and chain all local blocks", chain.groupId)
		chain_log.Debugf("<%s> height <%d>", chain.groupId, chain.group.Item.HighestHeight)
		chain_log.Debugf("<%s> block_id <%s>", chain.groupId, chain.group.Item.HighestBlockId)

		chain.syncer.SyncLocalBlock(blockId, nodename)
		topBlock, err := nodectx.GetDbMgr().GetBlock(chain.group.Item.HighestBlockId, false, nodename)
		if err != nil {
			chain_log.Warningf("Get top block error, blockId <%s>, <%s>", blockId, err.Error())
			return
		}
		if chain.syncer != nil {
			chain.syncer.SyncForward(topBlock)
		}
	}()

	return nil
}

func (chain *Chain) SyncBackward(blockId string, nodename string) error {
	chain_log.Debugf("<%s> SyncBackward called", chain.groupId)
	go func() {
		block, err := nodectx.GetDbMgr().GetBlock(blockId, false, nodename)
		if err != nil {
			chain_log.Warningf("Get block error, blockId <%s>, <%s>", blockId, err.Error())
			return
		}

		if chain.syncer != nil {
			chain.syncer.SyncBackward(block)
		}
	}()

	return nil
}

func (chain *Chain) StopSync() error {
	chain_log.Debugf("<%s> StopSync called", chain.groupId)
	if chain.syncer != nil {
		return chain.syncer.StopSync()
	}
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

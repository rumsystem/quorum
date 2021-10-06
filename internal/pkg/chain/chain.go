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

	chain.Consensus = NewMolasses(&MolassesProducer{NodeName: nodename})
	chain.Consensus.Producer().Init(group, chain.trxMgrs, chain.nodename)

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
	chain.updProducerList()

	//check if need to create molass producer
	chain.updProducer()
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

func (chain *Chain) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	chain_log.Infof("UpdAnnounce called")
	return chain.trxMgrs[chain.producerChannelId].SendAnnounceTrx(item)
}

func (chain *Chain) UpdBlkList(item *quorumpb.DenyUserItem) (string, error) {
	chain_log.Infof("UpdBlkList called")
	return chain.trxMgrs[chain.producerChannelId].SendUpdAuthTrx(item)
}

func (chain *Chain) UpdSchema(item *quorumpb.SchemaItem) (string, error) {
	chain_log.Infof("UpdSchema called")
	return chain.trxMgrs[chain.producerChannelId].SendUpdSchemaTrx(item)
}

func (chain *Chain) UpdProducer(item *quorumpb.ProducerItem) (string, error) {
	chain_log.Infof("UpdSchema called")
	return chain.trxMgrs[chain.producerChannelId].SendRegProducerTrx(item)
}

func (chain *Chain) PostToGroup(content proto.Message) (string, error) {
	chain_log.Infof("PostToGroup called to channel %s", chain.producerChannelId)
	return chain.trxMgrs[chain.producerChannelId].PostAny(content)
}

func (chain *Chain) GetProducerTrxMgr() *TrxMgr {
	return chain.trxMgrs[chain.producerChannelId]
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
		err := chain.UserAddBlock(block)
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

func (chain *Chain) UserAddBlock(block *quorumpb.Block) error {
	chain_log.Infof("userAddBlock called")

	//check if block is already in chain
	isSaved, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	if isSaved {
		return errors.New("Block already saved, ignore")
	}

	//check if block is in cache
	isCached, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, true, chain.nodename)
	if err != nil {
		return err
	}

	if isCached {
		return errors.New("Block already cached, ignore")
	}

	//Save block to cache
	err = nodectx.GetDbMgr().AddBlock(block, true, chain.nodename)
	if err != nil {
		return err
	}

	//check if parent of block exist
	parentExist, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		chain_log.Infof("Block Parent not exist, sync backward")
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetDbMgr().GetBlock(block.PrevBlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := IsBlockValid(block, parentBlock)
	if !valid {
		return err
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetDbMgr().GatherBlocksFromCache(block, true, chain.nodename)
	if err != nil {
		return err
	}

	//get all trxs from those blocks
	var trxs []*quorumpb.Trx
	trxs, err = chain.getAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply those trxs
	err = chain.userApplyTrx(trxs)
	if err != nil {
		return err
	}

	//move gathered blocks from cache to chain
	for _, block := range blocks {
		chain_log.Infof("Move block %s from cache to normal", block.BlockId)
		err := nodectx.GetDbMgr().AddBlock(block, false, chain.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetDbMgr().RmBlock(block.BlockId, true, chain.nodename)
		if err != nil {
			return err
		}
	}

	//calculate new height
	chain_log.Debugf("height before recal %d", chain.group.Item.HighestHeight)
	newHeight, newHighestBlockId, err := chain.recalChainHeight(blocks, chain.group.Item.HighestHeight)
	chain_log.Debugf("new height %d, new highest blockId %v", newHeight, newHighestBlockId)

	//if the new block is not highest block after recalculate, we need to "trim" the chain
	if newHeight < chain.group.Item.HighestHeight {

		//from parent of the new blocks, get all blocks not belong to the longest path
		resendBlocks, err := chain.getTrimedBlocks(blocks)
		if err != nil {
			return err
		}

		var resendTrxs []*quorumpb.Trx
		resendTrxs, err = chain.getMyTrxs(resendBlocks)

		if err != nil {
			return err
		}

		chain.updateResendCount(resendTrxs)
		err = chain.resendTrx(resendTrxs)
	}

	//update group info
	chain.group.Item.HighestHeight = newHeight
	chain.group.Item.HighestBlockId = newHighestBlockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	nodectx.GetDbMgr().UpdGroup(chain.group.Item)

	chain_log.Infof("userAddBlock ended")

	return nil
}

//addBlock for producer
func (chain *Chain) ProducerAddBlock(block *quorumpb.Block) error {
	chain_log.Infof("producerAddBlock called")

	//check if block is already in chain
	isSaved, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, false, chain.nodename)
	if err != nil {
		return err
	}
	if isSaved {
		return errors.New("Block already saved, ignore")
	}

	//check if block is in cache
	isCached, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, true, chain.nodename)
	if err != nil {
		return err
	}

	if isCached {
		return errors.New("Block already cached, ignore")
	}

	//Save block to cache
	err = nodectx.GetDbMgr().AddBlock(block, true, chain.nodename)
	if err != nil {
		return err
	}

	parentExist, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		chain_log.Infof("Block Parent not exist, sync backward")
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetDbMgr().GetBlock(block.PrevBlockId, false, chain.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := IsBlockValid(block, parentBlock)
	if !valid {
		return err
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetDbMgr().GatherBlocksFromCache(block, true, chain.nodename)
	if err != nil {
		return err
	}

	//get all trxs in those new blocks
	var trxs []*quorumpb.Trx
	trxs, err = chain.getAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply those trxs
	err = chain.producerApplyTrxs(trxs)
	if err != nil {
		return err
	}

	//move blocks from cache to normal
	for _, block := range blocks {
		chain_log.Infof("Move block %s from cache to normal", block.BlockId)
		err := nodectx.GetDbMgr().AddBlock(block, false, chain.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetDbMgr().RmBlock(block.BlockId, true, chain.nodename)
		if err != nil {
			return err
		}
	}

	chain_log.Infof("height before recal: %d", chain.group.Item.HighestHeight)
	newHeight, newHighestBlockId, err := chain.recalChainHeight(blocks, chain.group.Item.HighestHeight)
	chain_log.Infof("new height %d, new highest blockId %v", newHeight, newHighestBlockId)

	//update group info
	chain.group.Item.HighestHeight = newHeight
	chain.group.Item.HighestBlockId = newHighestBlockId
	chain.group.Item.LastUpdate = time.Now().UnixNano()
	nodectx.GetDbMgr().UpdGroup(chain.group.Item)

	return nil
}

//find the highest block from the block tree
func (chain *Chain) recalChainHeight(blocks []*quorumpb.Block, currentHeight int64) (int64, []string, error) {
	var highestBlockId []string
	newHeight := currentHeight
	for _, block := range blocks {
		blockHeight, err := nodectx.GetDbMgr().GetBlockHeight(block.BlockId, chain.nodename)
		if err != nil {
			return -1, highestBlockId, err
		}
		if blockHeight > newHeight {
			newHeight = blockHeight
			highestBlockId = nil
			highestBlockId = append(highestBlockId, block.BlockId)
		} else if blockHeight == newHeight {
			highestBlockId = append(highestBlockId, block.BlockId)
		} else {
			// do nothing
		}
	}
	return newHeight, highestBlockId, nil
}

//from root of the new block tree, get all blocks trimed when not belong to longest path
func (chain *Chain) getTrimedBlocks(blocks []*quorumpb.Block) ([]string, error) {
	var cache map[string]bool
	var longestPath []string
	var result []string

	cache = make(map[string]bool)

	err := chain.dfs(blocks, cache, longestPath)

	for _, blockId := range longestPath {
		if _, ok := cache[blockId]; !ok {
			result = append(result, blockId)
		}
	}

	return result, err
}

//TODO: need more test
func (chain *Chain) dfs(blocks []*quorumpb.Block, cache map[string]bool, result []string) error {
	for _, block := range blocks {
		if _, ok := cache[block.BlockId]; !ok {
			cache[block.BlockId] = true
			result = append(result, block.BlockId)
			subBlocks, err := nodectx.GetDbMgr().GetSubBlock(block.BlockId, chain.nodename)
			if err != nil {
				return err
			}
			err = chain.dfs(subBlocks, cache, result)
		}
	}
	return nil
}

//get all trx belongs to me from the block list
func (chain *Chain) getMyTrxs(blockIds []string) ([]*quorumpb.Trx, error) {
	chain_log.Infof("getMyTrxs called")
	var trxs []*quorumpb.Trx

	for _, blockId := range blockIds {
		block, err := nodectx.GetDbMgr().GetBlock(blockId, false, chain.nodename)
		if err != nil {
			chain_log.Warnf(err.Error())
			continue
		}

		for _, trx := range block.Trxs {
			if trx.SenderPubkey == chain.group.Item.UserSignPubkey {
				trxs = append(trxs, trx)
			}
		}
	}
	return trxs, nil
}

//get all trx from the block list
func (chain *Chain) getAllTrxs(blocks []*quorumpb.Block) ([]*quorumpb.Trx, error) {
	chain_log.Infof("getAllTrxs called")
	var trxs []*quorumpb.Trx
	for _, block := range blocks {
		for _, trx := range block.Trxs {
			trxs = append(trxs, trx)
		}
	}
	return trxs, nil
}

//update resend count (+1) for all trxs
func (chain *Chain) updateResendCount(trxs []*quorumpb.Trx) ([]*quorumpb.Trx, error) {
	chain_log.Infof("updateResendCount called")
	for _, trx := range trxs {
		trx.ResendCount++
	}
	return trxs, nil
}

//resend all trx in the list
func (chain *Chain) resendTrx(trxs []*quorumpb.Trx) error {
	chain_log.Infof("resendTrx")
	for _, trx := range trxs {
		chain_log.Infof("Resend Trx %s", trx.TrxId)
		chain.trxMgrs[chain.producerChannelId].ResendTrx(trx)
	}
	return nil
}

func (chain *Chain) userApplyTrx(trxs []*quorumpb.Trx) error {
	chain_log.Infof("applyTrxs called")
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetDbMgr().IsTrxExist(trx.TrxId, chain.nodename)
		if err != nil {
			chain_log.Infof(err.Error())
			continue
		}

		if isExist {
			chain_log.Infof("Trx %s existed, update trx only", trx.TrxId)
			nodectx.GetDbMgr().AddTrx(trx)
			continue
		}

		//new trx, apply it
		if trx.Type == quorumpb.TrxType_POST && chain.group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(chain.group.Item.UserEncryptPubkey, trx.Data)
			if err != nil {
				return err
			}
			trx.Data = decryptData
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

			trx.Data = decryptData
		}

		chain_log.Infof("try apply trx %s", trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Infof("Apply POST trx")
			nodectx.GetDbMgr().AddPost(trx, chain.nodename)
		case quorumpb.TrxType_AUTH:
			chain_log.Infof("Apply AUTH trx")
			nodectx.GetDbMgr().UpdateBlkListItem(trx, chain.nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Infof("Apply PRODUCER Trx")
			nodectx.GetDbMgr().UpdateProducer(trx, chain.nodename)
			chain.updProducerList()
			chain.updProducer()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Infof("Apply ANNOUNCE trx")
			nodectx.GetDbMgr().UpdateAnnounce(trx, chain.nodename)
		case quorumpb.TrxType_SCHEMA:
			chain_log.Infof("Apply SCHEMA trx ")
			nodectx.GetDbMgr().UpdateSchema(trx, chain.nodename)
		default:
			chain_log.Infof("Unsupported msgType %s", trx.Type)
		}

		//save trx to db
		nodectx.GetDbMgr().AddTrx(trx, chain.nodename)
	}

	return nil
}

func (chain *Chain) producerApplyTrxs(trxs []*quorumpb.Trx) error {
	chain_log.Infof("applyTrxs called")
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetDbMgr().IsTrxExist(trx.TrxId, chain.nodename)
		if err != nil {
			chain_log.Infof(err.Error())
			continue
		}

		if isExist {
			chain_log.Infof("Trx %s existed, update trx", trx.TrxId)
			nodectx.GetDbMgr().AddTrx(trx)
			continue
		}

		//make deep copy trx to avoid modify data of the original trx
		var copiedTrx *quorumpb.Trx
		copiedTrx = &quorumpb.Trx{}
		copiedTrx.TrxId = trx.TrxId
		copiedTrx.Type = trx.Type
		copiedTrx.GroupId = trx.GroupId
		copiedTrx.TimeStamp = trx.TimeStamp
		copiedTrx.Version = trx.Version
		copiedTrx.Expired = trx.Expired
		copiedTrx.ResendCount = trx.ResendCount
		copiedTrx.Nonce = trx.Nonce
		copiedTrx.SenderPubkey = trx.SenderPubkey
		copiedTrx.SenderSign = trx.SenderSign

		if trx.Type == quorumpb.TrxType_POST && chain.group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			//just try decrypt it, if failed, save the original encrypted data
			//the reason for that is, for private group, before owner add producer, owner is the only producer,
			//since owner also needs to show POST data, and all announced user will encrypt for owner pubkey
			//owner can actually decrypt POST
			//for other producer, they can not decrpyt POST
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(chain.group.Item.UserEncryptPubkey, trx.Data)
			if err != nil {
				copiedTrx.Data = trx.Data
			} else {
				copiedTrx.Data = decryptData
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

			copiedTrx.Data = decryptData
		}

		chain_log.Infof("try apply trx %s", trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_POST:
			chain_log.Infof("Apply POST trx")
			nodectx.GetDbMgr().AddPost(copiedTrx, chain.nodename)
		case quorumpb.TrxType_AUTH:
			chain_log.Infof("Apply AUTH trx")
			nodectx.GetDbMgr().UpdateBlkListItem(copiedTrx, chain.nodename)
		case quorumpb.TrxType_PRODUCER:
			chain_log.Infof("Apply PRODUCER Trx")
			nodectx.GetDbMgr().UpdateProducer(copiedTrx, chain.nodename)
			chain.updProducerList()
			chain.updProducer()
		case quorumpb.TrxType_ANNOUNCE:
			chain_log.Infof("Apply ANNOUNCE trx")
			nodectx.GetDbMgr().UpdateAnnounce(copiedTrx, chain.nodename)
		case quorumpb.TrxType_SCHEMA:
			chain_log.Infof("Apply SCHEMA trx ")
			nodectx.GetDbMgr().UpdateSchema(copiedTrx, chain.nodename)
		default:
			chain_log.Infof("Unsupported msgType %s", copiedTrx.Type)
		}

		//save trx to db
		nodectx.GetDbMgr().AddTrx(copiedTrx, chain.nodename)
	}

	return nil
}

func (chain *Chain) updProducerList() {
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

func (chain *Chain) updProducer() {
	if _, ok := chain.ProducerPool[chain.group.Item.UserSignPubkey]; ok {
		//Yes, I am producer, create censensus and group producer
		chain_log.Infof("Create and initial molasses producer")
		chain.Consensus = NewMolasses(&MolassesProducer{})
		chain.Consensus.Producer().Init(chain.group, chain.trxMgrs, chain.nodename)
	} else {
		chain_log.Infof("Set molasses producer to nil")
		chain.Consensus = nil
	}
}

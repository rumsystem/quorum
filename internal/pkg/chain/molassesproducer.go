package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"google.golang.org/protobuf/proto"
)

var molaproducer_log = logging.Logger("producer")

const PRODUCE_TIMER time.Duration = 5 //5s
const MERGE_TIMER time.Duration = 5   //5s

type ProducerStatus int

const (
	StatusIdle ProducerStatus = iota
	StatusMerging
	StatusProducing
)

type MolassesProducer struct {
	grp          *Group
	blockPool    map[string]*quorumpb.Block
	trxPool      map[string]*quorumpb.Trx
	trxMgr       map[string]*TrxMgr
	status       ProducerStatus
	NodeName     string
	ProduceTimer *time.Timer
	ProduceDone  chan bool
	statusmu     sync.RWMutex
	//trxpoolmu    sync.RWMutex
}

func (producer *MolassesProducer) Init(grp *Group, trxMgr map[string]*TrxMgr, nodeName string) {
	molaproducer_log.Infof("Init called")
	//producer.grp = grp
	producer.trxMgr = trxMgr
	producer.trxPool = make(map[string]*quorumpb.Trx)
	producer.blockPool = make(map[string]*quorumpb.Block)
	producer.status = StatusIdle
	producer.NodeName = nodeName
}

// Add trx to trx pool
func (producer *MolassesProducer) AddTrx(trx *quorumpb.Trx) {
	molaproducer_log.Infof("Molasses AddTrx called, add trx %s", trx.TrxId)
	if producer.isSyncing() {
		return
	}

	//producer.trxpoolmu.Lock()
	//defer producer.trxpoolmu.Unlock()
	producer.trxPool[trx.TrxId] = trx

	//launch produce
	if producer.status == StatusIdle {
		go producer.startProduceBlock()
	}
}

func (producer *MolassesProducer) startProduceBlock() {
	molaproducer_log.Infof("Molasses startProduceBlock called on %s", producer.NodeName)
	producer.ProduceTimer = time.NewTimer(PRODUCE_TIMER * time.Second)
	producer.statusmu.Lock()
	producer.status = StatusProducing
	molaproducer_log.Infof("%s set StatusProducing", producer.NodeName)
	defer func() {
		producer.ProduceTimer.Stop()
		producer.status = StatusIdle
		molaproducer_log.Infof("%s set StatusIdle", producer.NodeName)
		producer.statusmu.Unlock()
	}()

	t := <-producer.ProduceTimer.C
	molaproducer_log.Infof("%s Producer wait done at %s", producer.NodeName, t.UTC().String())
	producer.produceBlock()
}

func (producer *MolassesProducer) produceBlock() {
	molaproducer_log.Infof("produceBlock called")

	//for multi longest chains, sort and use the first BlockId as the winner.
	sort.Strings(producer.grp.Item.HighestBlockId)
	highestBlockIdWinner := producer.grp.Item.HighestBlockId[0]

	topBlock, err := nodectx.GetDbMgr().GetBlock(highestBlockIdWinner, false, producer.NodeName)
	if err != nil {
		molaproducer_log.Infof(err.Error())
		return
	}

	//Don't lock trx pool, just package what ever you have at this moment
	//package all trx
	//producer.trxpoolmu.Lock()
	//defer producer.trxpoolmu.Unlock()
	molaproducer_log.Infof("package %d trxs", len(producer.trxPool))
	trxs := make([]*quorumpb.Trx, 0, len(producer.trxPool))
	for key, value := range producer.trxPool {
		copyValue := value
		trxs = append(trxs, copyValue)
		//remove trx from pool
		delete(producer.trxPool, key)
	}

	//create block
	pubkeyBytes, err := p2pcrypto.ConfigDecodeKey(producer.grp.Item.UserSignPubkey)
	if err != nil {
		molaproducer_log.Infof(err.Error())
		return
	}

	newBlock, err := CreateBlock(topBlock, trxs, pubkeyBytes, producer.NodeName)
	if err != nil {
		molaproducer_log.Errorf(err.Error())
		return
	}

	//CREATE AND BROADCAST NEW BLOCK BY USING BLOCK_PRODUCED MSG ON PRODUCER CHANNEL
	producerTrxMgr := producer.grp.ChainCtx.GetProducerTrxMgr()
	producerTrxMgr.SendBlockProduced(newBlock)

	molaproducer_log.Infof("%s Produce done, wait for merge", producer.NodeName)
	//producer.trxPool = make(map[string]*quorumpb.Trx)
}

func (producer *MolassesProducer) AddBlockToPool(block *quorumpb.Block) {
	molaproducer_log.Infof("AddBlockToPool called")
	if producer.isSyncing() {
		return
	}
	producer.blockPool[block.BlockId] = block
}

func (producer *MolassesProducer) AddProducedBlock(trx *quorumpb.Trx) error {
	molaproducer_log.Infof("AddProducedBlock called")
	if producer.isSyncing() {
		return nil
	}

	ciperKey, err := hex.DecodeString(producer.grp.Item.CipherKey)
	if err != nil {
		return err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return err
	}

	block := &quorumpb.Block{}
	if err := proto.Unmarshal(decryptData, block); err != nil {
		return err
	}

	molaproducer_log.Infof("%s Add Block to Pool and set merge timer to 5s", producer.NodeName)
	producer.AddBlockToPool(block)

	//if merge already started
	if producer.status == StatusMerging {
		return nil
	}

	go producer.startMergeBlock()
	return nil
}

func (producer *MolassesProducer) startMergeBlock() error {
	molaproducer_log.Infof("startMergeBlock called")
	producer.statusmu.Lock()
	producer.status = StatusMerging
	molaproducer_log.Infof("%s set StatusMerging", producer.NodeName)
	defer func() {
		molaproducer_log.Infof("%s set StatusIdle", producer.NodeName)
		producer.status = StatusIdle
		producer.statusmu.Unlock()

		if len(producer.trxPool) != 0 {
			molaproducer_log.Infof("%s start produce block", producer.NodeName)
			producer.startProduceBlock()
		}
	}()

	mergeTimer := time.NewTimer(MERGE_TIMER * time.Second)
	t := <-mergeTimer.C
	molaproducer_log.Infof("%s merge timer ticker...%s", producer.NodeName, t.UTC().String())

	candidateBlkid := ""
	var oHash []byte
	for _, blk := range producer.blockPool {
		nHash := sha256.Sum256(blk.Signature)
		//comparing two hash bytes lexicographically
		if bytes.Compare(oHash[:], nHash[:]) == -1 { //-1 means ohash < nhash, and we want keep the larger one
			candidateBlkid = blk.BlockId
			oHash = nHash[:]
		}
	}

	molaproducer_log.Debugf("Candidate block decided, block Id : %s", candidateBlkid)
	err := producer.AddBlock(producer.blockPool[candidateBlkid])

	if err != nil {
		molaproducer_log.Errorf("save block %s error %s", candidateBlkid, err)
	} else {
		molaproducer_log.Debugf("saved, HighestHeight: %d Highestblkid: %s", producer.grp.Item.HighestHeight, producer.grp.Item.HighestBlockId)
		molaproducer_log.Infof("Send new block out")
		err := producer.trxMgr[producer.grp.ChainCtx.userChannelId].SendBlock(producer.blockPool[candidateBlkid])
		if err != nil {
			molaproducer_log.Warnf(err.Error())
		}
	}

	molaproducer_log.Infof("%s Merge done, clear all blocks", producer.NodeName)
	producer.blockPool = make(map[string]*quorumpb.Block)

	return nil
}

func (producer *MolassesProducer) GetBlockForward(trx *quorumpb.Trx) error {
	molaproducer_log.Infof("GetBlockForward called")

	var reqBlockItem quorumpb.ReqBlock
	ciperKey, err := hex.DecodeString(producer.grp.Item.CipherKey)
	if err != nil {
		return err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return err
	}

	//check if requester is in group block list
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey)

	if isBlocked {
		molaproducer_log.Warning("user is blocked by group owner")
		err := errors.New("user auth failed")
		return err
	}

	subBlocks, err := nodectx.GetDbMgr().GetSubBlock(reqBlockItem.BlockId, producer.NodeName)

	if err != nil {
		return err
	}

	if len(subBlocks) != 0 {
		for _, block := range subBlocks {
			molaproducer_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
			err := producer.trxMgr[producer.grp.ChainCtx.producerChannelId].SendReqBlockResp(&reqBlockItem, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
			if err != nil {
				molaproducer_log.Warnf(err.Error())
			}
		}
		return nil
	} else {
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		//set producer pubkey of empty block
		emptyBlock.BlockId = guuid.New().String()
		emptyBlock.ProducerPubKey = producer.grp.Item.UserSignPubkey
		molaproducer_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)")
		return producer.trxMgr[producer.grp.ChainCtx.producerChannelId].SendReqBlockResp(&reqBlockItem, emptyBlock, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
	}
}

func (producer *MolassesProducer) GetBlockBackward(trx *quorumpb.Trx) error {
	molaproducer_log.Infof("GetBlockBackward called")

	var reqBlockItem quorumpb.ReqBlock

	ciperKey, err := hex.DecodeString(producer.grp.Item.CipherKey)
	if err != nil {
		return err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return err
	}

	//check if requester is in group block list
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey)

	if isBlocked {
		molaproducer_log.Warning("user is blocked by group owner")
		err := errors.New("user auth failed")
		return err
	}

	isExist, err := nodectx.GetDbMgr().IsBlockExist(reqBlockItem.BlockId, false, producer.NodeName)
	if err != nil {
		return err
	} else if !isExist {
		return errors.New("Block not exist")
	}

	block, err := nodectx.GetDbMgr().GetBlock(reqBlockItem.BlockId, false, producer.NodeName)
	if err != nil {
		return err
	}

	isParentExit, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, producer.NodeName)
	if err != nil {
		return err
	}

	if isParentExit {
		molaproducer_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
		parentBlock, err := nodectx.GetDbMgr().GetParentBlock(reqBlockItem.BlockId, producer.NodeName)
		if err != nil {
			return err
		}
		return producer.trxMgr[producer.grp.ChainCtx.producerChannelId].SendReqBlockResp(&reqBlockItem, parentBlock, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
	} else {
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		emptyBlock.BlockId = guuid.New().String()
		emptyBlock.ProducerPubKey = producer.grp.Item.UserSignPubkey
		molaproducer_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)")
		return producer.trxMgr[producer.grp.ChainCtx.producerChannelId].SendReqBlockResp(&reqBlockItem, emptyBlock, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
	}
}

func (producer *MolassesProducer) isSyncing() bool {
	if producer.grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD ||
		producer.grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		molaproducer_log.Infof("Producer in syncing")
		return true
	}

	return false
}

//addBlock for producer
func (producer *MolassesProducer) AddBlock(block *quorumpb.Block) error {
	molaproducer_log.Infof("producerAddBlock called")

	//check if block is already in chain
	isSaved, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, false, producer.NodeName)
	if err != nil {
		return err
	}
	if isSaved {
		return errors.New("Block already saved, ignore")
	}

	//check if block is in cache
	isCached, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, true, producer.NodeName)
	if err != nil {
		return err
	}

	if isCached {
		return errors.New("Block already cached, ignore")
	}

	//Save block to cache
	err = nodectx.GetDbMgr().AddBlock(block, true, producer.NodeName)
	if err != nil {
		return err
	}

	parentExist, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, producer.NodeName)
	if err != nil {
		return err
	}

	if !parentExist {
		molaproducer_log.Infof("Block Parent not exist, sync backward")
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetDbMgr().GetBlock(block.PrevBlockId, false, producer.NodeName)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := IsBlockValid(block, parentBlock)
	if !valid {
		return err
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetDbMgr().GatherBlocksFromCache(block, true, producer.NodeName)
	if err != nil {
		return err
	}

	//get all trxs in those new blocks
	var trxs []*quorumpb.Trx
	trxs, err = GetAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply those trxs
	err = producer.applyTrxs(trxs)
	if err != nil {
		return err
	}

	//move blocks from cache to normal
	for _, block := range blocks {
		molaproducer_log.Infof("Move block %s from cache to normal", block.BlockId)
		err := nodectx.GetDbMgr().AddBlock(block, false, producer.NodeName)
		if err != nil {
			return err
		}

		err = nodectx.GetDbMgr().RmBlock(block.BlockId, true, producer.NodeName)
		if err != nil {
			return err
		}
	}

	molaproducer_log.Infof("height before recal: %d", producer.grp.Item.HighestHeight)
	newHeight, newHighestBlockId, err := RecalChainHeight(blocks, producer.grp.Item.HighestHeight, producer.NodeName)
	molaproducer_log.Infof("new height %d, new highest blockId %v", newHeight, newHighestBlockId)

	return producer.grp.ChainCtx.group.ChainCtx.UpdChainInfo(newHeight, newHighestBlockId)
}

func (producer *MolassesProducer) applyTrxs(trxs []*quorumpb.Trx) error {
	molaproducer_log.Infof("applyTrxs called")
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetDbMgr().IsTrxExist(trx.TrxId, producer.NodeName)
		if err != nil {
			molaproducer_log.Infof(err.Error())
			continue
		}

		if isExist {
			molaproducer_log.Infof("Trx %s existed, update trx", trx.TrxId)
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

		if trx.Type == quorumpb.TrxType_POST && producer.grp.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			//just try decrypt it, if failed, save the original encrypted data
			//the reason for that is, for private group, before owner add producer, owner is the only producer,
			//since owner also needs to show POST data, and all announced user will encrypt for owner pubkey
			//owner can actually decrypt POST
			//for other producer, they can not decrpyt POST
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(producer.grp.Item.UserEncryptPubkey, trx.Data)
			if err != nil {
				copiedTrx.Data = trx.Data
			} else {
				copiedTrx.Data = decryptData
			}

		} else {
			//decode trx data
			ciperKey, err := hex.DecodeString(producer.grp.Item.CipherKey)
			if err != nil {
				return err
			}

			decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
			if err != nil {
				return err
			}

			copiedTrx.Data = decryptData
		}

		molaproducer_log.Infof("try apply trx %s", trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_POST:
			molaproducer_log.Infof("Apply POST trx")
			nodectx.GetDbMgr().AddPost(copiedTrx, producer.NodeName)
		case quorumpb.TrxType_AUTH:
			molaproducer_log.Infof("Apply AUTH trx")
			nodectx.GetDbMgr().UpdateBlkListItem(copiedTrx, producer.NodeName)
		case quorumpb.TrxType_PRODUCER:
			molaproducer_log.Infof("Apply PRODUCER Trx")
			nodectx.GetDbMgr().UpdateProducer(copiedTrx, producer.NodeName)
			producer.grp.ChainCtx.UpdProducerList()
			producer.grp.ChainCtx.UpdProducer()
		case quorumpb.TrxType_ANNOUNCE:
			molaproducer_log.Infof("Apply ANNOUNCE trx")
			nodectx.GetDbMgr().UpdateAnnounce(copiedTrx, producer.NodeName)
		case quorumpb.TrxType_SCHEMA:
			molaproducer_log.Infof("Apply SCHEMA trx ")
			nodectx.GetDbMgr().UpdateSchema(copiedTrx, producer.NodeName)
		default:
			molaproducer_log.Infof("Unsupported msgType %s", copiedTrx.Type)
		}

		//save trx to db
		nodectx.GetDbMgr().AddTrx(copiedTrx, producer.NodeName)
	}

	return nil
}

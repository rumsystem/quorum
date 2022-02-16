package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var molaproducer_log = logging.Logger("producer")

const PRODUCE_TIMER time.Duration = 5     //5s
const MERGE_TIMER time.Duration = 5       //5s
const CLOSE_CONN_TIMER time.Duration = 20 //20s

const TRXS_TOTAL_SIZE int = 900 * 1024

type ProducerStatus int

const (
	StatusIdle ProducerStatus = iota
	StatusMerging
	StatusProducing
)

type MolassesProducer struct {
	grpItem           *quorumpb.GroupItem
	blockPool         map[string]*quorumpb.Block
	trxPool           sync.Map
	trxMgr            map[string]*TrxMgr
	syncConnTimerPool map[string]*time.Timer
	status            ProducerStatus
	ProduceTimer      *time.Timer
	ProduceDone       chan bool
	statusmu          sync.RWMutex
	nodename          string
	cIface            ChainMolassesIface
	groupId           string
}

func (producer *MolassesProducer) Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface) {
	molaproducer_log.Debug("Init called")
	producer.grpItem = item
	producer.cIface = iface
	producer.blockPool = make(map[string]*quorumpb.Block)
	//producer.trxPool = make(map[string]*quorumpb.Trx)
	producer.trxMgr = make(map[string]*TrxMgr)
	producer.syncConnTimerPool = make(map[string]*time.Timer)
	producer.status = StatusIdle
	producer.nodename = nodename
	producer.groupId = item.GroupId

	molaproducer_log.Infof("<%s> producer created", producer.groupId)
}

// Add trx to trx pool
func (producer *MolassesProducer) AddTrx(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> AddTrx called", producer.groupId)

	//check if trx sender is in group block list
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey, producer.nodename)

	if isBlocked {
		molaproducer_log.Debugf("<%s> user <%s> is blocked", producer.groupId, trx.SenderPubkey)
		return
	} else {
		molaproducer_log.Debugf("<%s> user <%s> is not blocked", producer.groupId, trx.SenderPubkey)
	}

	if producer.cIface.IsSyncerReady() {
		return
	}

	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	producer.trxPool.Store(trx.TrxId, trx)

	molaproducer_log.Debugf("*************************************")
	producer.trxPool.Range(func(key, value interface{}) bool {
		trxId, _ := key.(string)
		molaproducer_log.Debugf("key <%s>", trxId)
		return true
	})
	molaproducer_log.Debugf("*************************************")

	if producer.status == StatusIdle {
		go producer.startProduceBlock()
	}
}

func (producer *MolassesProducer) startProduceBlock() {
	molaproducer_log.Debugf("<%s> startProduceBlock called", producer.groupId)
	producer.ProduceTimer = time.NewTimer(PRODUCE_TIMER * time.Second)
	producer.statusmu.Lock()
	producer.status = StatusProducing
	molaproducer_log.Debugf("<%s> set StatusProducing", producer.groupId)
	defer func() {
		producer.ProduceTimer.Stop()
		producer.status = StatusIdle
		molaproducer_log.Debugf("<%s> set StatusIdle", producer.groupId)
		producer.statusmu.Unlock()
	}()

	t := <-producer.ProduceTimer.C
	molaproducer_log.Debugf("<%s> producer wait done at (%s)", producer.groupId, t.UTC().String())
	producer.produceBlock()
}

func (producer *MolassesProducer) produceBlock() {
	molaproducer_log.Debugf("<%s> produceBlock called", producer.groupId)
	topBlock, err := nodectx.GetDbMgr().GetBlock(producer.grpItem.HighestBlockId, false, producer.nodename)
	if err != nil {
		molaproducer_log.Info(err.Error())
		return
	}

	//Don't lock trx pool, just package what ever you have at this moment
	//package all trx
	var trxs []*quorumpb.Trx
	totalSizeBytes := 0
	totalTrx := 0

	producer.trxPool.Range(func(k, v interface{}) bool {
		trxId, _ := k.(string)
		trx, _ := v.(*quorumpb.Trx)

		encodedcontent, _ := quorumpb.ContentToBytes(trx)
		totalSizeBytes += binary.Size(encodedcontent)

		if totalSizeBytes < TRXS_TOTAL_SIZE {
			trxs = append(trxs, trx)
			//remove trx from pool
			producer.trxPool.Delete(trxId)
			totalTrx++

			return true
		}

		return false
	})

	molaproducer_log.Debugf("*************after package***************")
	producer.trxPool.Range(func(key, value interface{}) bool {
		trxId, _ := key.(string)
		molaproducer_log.Debugf("key <%s>", trxId)
		return true
	})
	molaproducer_log.Debugf("*************************************")

	molaproducer_log.Debugf("<%s> package <%d> trxs, size <%d>", producer.groupId, totalTrx, totalSizeBytes)

	//create block
	pubkeyBytes, err := p2pcrypto.ConfigDecodeKey(producer.grpItem.UserSignPubkey)
	if err != nil {
		molaproducer_log.Debug(err.Error())
		return
	}

	newBlock, err := CreateBlock(topBlock, trxs, pubkeyBytes, producer.nodename)
	if err != nil {
		molaproducer_log.Errorf("<%s> create block error", producer.groupId)
		molaproducer_log.Errorf(err.Error())
		return
	}

	//CREATE AND BROADCAST NEW BLOCK BY USING BLOCK_PRODUCED MSG ON PRODUCER CHANNEL
	molaproducer_log.Debugf("<%s> broadcast produced block", producer.groupId)
	producer.cIface.GetProducerTrxMgr().SendBlockProduced(newBlock)
	molaproducer_log.Debugf("<%s> produce done, wait for merge", producer.groupId)
}

func (producer *MolassesProducer) AddBlockToPool(block *quorumpb.Block) {
	molaproducer_log.Debugf("<%s> AddBlockToPool called", producer.groupId)
	if producer.cIface.IsSyncerReady() {
		return
	}
	producer.blockPool[block.BlockId] = block
}

func (producer *MolassesProducer) AddProducedBlock(trx *quorumpb.Trx) error {
	molaproducer_log.Debugf("<%s> AddProducedBlock called", producer.groupId)
	if producer.cIface.IsSyncerReady() {
		return nil
	}

	ciperKey, err := hex.DecodeString(producer.grpItem.CipherKey)
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

	molaproducer_log.Debugf("<%s> add produced block to Pool", producer.groupId)
	producer.AddBlockToPool(block)

	//if merge already started
	if producer.status == StatusMerging {
		return nil
	}

	producer.statusmu.Lock()
	producer.status = StatusMerging
	molaproducer_log.Debugf("<%s> set StatusMerging", producer.groupId)
	go producer.startMergeBlock()

	return nil
}

func (producer *MolassesProducer) startMergeBlock() error {
	molaproducer_log.Debugf("<%s> startMergeBlock called", producer.groupId)

	defer func() {
		molaproducer_log.Infof("<%s> set StatusIdle", producer.groupId)
		producer.status = StatusIdle
		producer.statusmu.Unlock()

		//since sync.map don't have len(), count manually
		var count uint
		producer.trxPool.Range(func(key interface{}, value interface{}) bool {
			count++
			return true
		})

		if count != 0 {
			molaproducer_log.Debugf("<%s> start produce block", producer.groupId)
			producer.startProduceBlock()
		}
	}()
	molaproducer_log.Debugf("<%s> set merge timer to <%d>s", producer.groupId, MERGE_TIMER)
	mergeTimer := time.NewTimer(MERGE_TIMER * time.Second)
	t := <-mergeTimer.C
	molaproducer_log.Debugf("<%s> merge timer ticker...<%s>", producer.groupId, t.UTC().String())

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

	molaproducer_log.Debugf("<%s> candidate block decided, block Id : %s", producer.groupId, candidateBlkid)

	surfix := ""
	if producer.blockPool[candidateBlkid].ProducerPubKey == producer.grpItem.OwnerPubKey {
		surfix = "OWNER"
	} else {
		surfix = "PRODUCER"
	}

	molaproducer_log.Debugf("<%s> winner <%s> (%s)", producer.groupId, producer.blockPool[candidateBlkid].ProducerPubKey, surfix)
	err := producer.AddBlock(producer.blockPool[candidateBlkid])

	if err != nil {
		molaproducer_log.Errorf("<%s> save block <%s> error <%s>", producer.groupId, candidateBlkid, err)
		if err.Error() == "PARENT_NOT_EXIST" {
			molaproducer_log.Debugf("<%s> parent not found, sync backward for missing blocks from <%s>", producer.groupId, candidateBlkid, err)
			producer.cIface.SyncBackward(producer.blockPool[candidateBlkid])
		}
	} else {
		molaproducer_log.Debugf("<%s> block saved", producer.groupId)
		//check if I am the winner
		if producer.blockPool[candidateBlkid].ProducerPubKey == producer.grpItem.UserSignPubkey {
			molaproducer_log.Debugf("<%s> winner send new block out", producer.groupId)
			err := producer.cIface.GetUserTrxMgr().SendBlock(producer.blockPool[candidateBlkid])
			if err != nil {
				molaproducer_log.Warnf("<%s> <%s>", producer.groupId, err.Error())
			}
		}
	}

	molaproducer_log.Debugf("<%s> merge done", producer.groupId)
	producer.blockPool = make(map[string]*quorumpb.Block)

	return nil
}

func (producer *MolassesProducer) GetBlockForward(trx *quorumpb.Trx) error {
	molaproducer_log.Debugf("<%s> GetBlockForward called", producer.groupId)

	var reqBlockItem quorumpb.ReqBlock
	ciperKey, err := hex.DecodeString(producer.grpItem.CipherKey)
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
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey, producer.nodename)

	if isBlocked {
		molaproducer_log.Debugf("<%s> user <%s> is blocked", producer.groupId, trx.SenderPubkey)
		return nil
	}

	subBlocks, err := nodectx.GetDbMgr().GetSubBlock(reqBlockItem.BlockId, producer.nodename)

	if err != nil {
		return err
	}

	channelId := SYNC_CHANNEL_PREFIX + producer.grpItem.GroupId + "_" + reqBlockItem.UserId
	trxMgr, _ := producer.getSyncConn(channelId)

	if len(subBlocks) != 0 {
		for _, block := range subBlocks {
			molaproducer_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", producer.groupId)
			err := trxMgr.SendReqBlockResp(&reqBlockItem, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
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
		emptyBlock.ProducerPubKey = producer.grpItem.UserSignPubkey
		molaproducer_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", producer.groupId)
		return trxMgr.SendReqBlockResp(&reqBlockItem, emptyBlock, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
	}
}

func (producer *MolassesProducer) GetBlockBackward(trx *quorumpb.Trx) error {
	molaproducer_log.Debugf("<%s> GetBlockBackward called", producer.groupId)

	var reqBlockItem quorumpb.ReqBlock

	ciperKey, err := hex.DecodeString(producer.grpItem.CipherKey)
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
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey, producer.nodename)

	if isBlocked {
		molaproducer_log.Debugf("<%s> user <%s> is blocked", producer.groupId, trx.SenderPubkey)
		return nil
	}

	isExist, err := nodectx.GetDbMgr().IsBlockExist(reqBlockItem.BlockId, false, producer.nodename)
	if err != nil {
		return err
	} else if !isExist {
		return errors.New("Block not exist")
	}

	block, err := nodectx.GetDbMgr().GetBlock(reqBlockItem.BlockId, false, producer.nodename)
	if err != nil {
		return err
	}

	isParentExit, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, producer.nodename)
	if err != nil {
		return err
	}

	channelId := SYNC_CHANNEL_PREFIX + producer.grpItem.GroupId + "_" + reqBlockItem.UserId
	trxMgr, _ := producer.getSyncConn(channelId)

	if isParentExit {
		molaproducer_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", producer.groupId)
		parentBlock, err := nodectx.GetDbMgr().GetParentBlock(reqBlockItem.BlockId, producer.nodename)
		if err != nil {
			return err
		}
		return trxMgr.SendReqBlockResp(&reqBlockItem, parentBlock, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
	} else {
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		emptyBlock.BlockId = guuid.New().String()
		emptyBlock.ProducerPubKey = producer.grpItem.UserSignPubkey
		molaproducer_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", producer.groupId)
		return trxMgr.SendReqBlockResp(&reqBlockItem, emptyBlock, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
	}
}

func (producer *MolassesProducer) HandleAskPeerId(trx *quorumpb.Trx) error {
	molaproducer_log.Debugf("<%s> HandleAskPeerId called", producer.groupId)
	var reqItem quorumpb.AskPeerId
	ciperKey, err := hex.DecodeString(producer.grpItem.CipherKey)
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
	isBlocked, _ := nodectx.GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey, producer.nodename)

	if isBlocked {
		molaproducer_log.Debugf("<%s> user <%s> is blocked", producer.groupId, trx.SenderPubkey)
		return nil
	}

	var respItem quorumpb.AskPeerIdResp
	respItem = quorumpb.AskPeerIdResp{}

	respItem.GroupId = producer.groupId
	respItem.RespPeerId = nodectx.GetNodeCtx().PeerId.Pretty()
	respItem.RespPeerPubkey = producer.grpItem.UserSignPubkey

	trxMgr := producer.cIface.GetProducerTrxMgr()

	return trxMgr.SendAskPeerIdResp(&respItem)
}

func (producer *MolassesProducer) getSyncConn(channelId string) (*TrxMgr, error) {
	molaproducer_log.Debugf("<%s> getSyncConn called", producer.groupId)

	var syncTrxMgr *TrxMgr

	if _, ok := producer.trxMgr[channelId]; ok {
		syncTrxMgr = producer.trxMgr[channelId]

		//reset timer
		molaproducer_log.Debugf("<%s> reset timer for sync channel <%s>", producer.groupId, channelId)
		timer := producer.syncConnTimerPool[channelId]
		timer.Stop()
		timer.Reset(CLOSE_CONN_TIMER * time.Second)
	} else {
		molaproducer_log.Debugf("<%s> create sync channel <%s>", producer.groupId, channelId)
		syncPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(channelId, producer.cIface.GetChainCtx())
		syncTrxMgr = &TrxMgr{}
		syncTrxMgr.Init(producer.grpItem, syncPsconn)
		producer.trxMgr[channelId] = syncTrxMgr

		molaproducer_log.Debugf("<%s> create close_conn timer for sync channel <%s>", producer.groupId, channelId)
		timer := time.AfterFunc(CLOSE_CONN_TIMER*time.Second, func() {
			if _, ok := producer.trxMgr[channelId]; ok {
				molaproducer_log.Debugf("<%s> time up, close sync channel <%s>", producer.groupId, channelId)
				//syncTrxMgr.LeaveChannel(channelId)
				nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(channelId)
				delete(producer.trxMgr, channelId)
			}
		})
		producer.syncConnTimerPool[channelId] = timer
	}

	return syncTrxMgr, nil
}

//addBlock for producer
func (producer *MolassesProducer) AddBlock(block *quorumpb.Block) error {
	molaproducer_log.Debugf("<%s> AddBlock called", producer.groupId)

	//check if block is already in chain
	isSaved, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, false, producer.nodename)
	if err != nil {
		return err
	}
	if isSaved {
		return errors.New("Block already saved, ignore")
	}

	//check if block is in cache
	isCached, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, true, producer.nodename)
	if err != nil {
		return err
	}

	if isCached {
		molaproducer_log.Debugf("<%s> Block cached, update block", producer.groupId)
	}

	//Save block to cache
	err = nodectx.GetDbMgr().AddBlock(block, true, producer.nodename)
	if err != nil {
		return err
	}

	parentExist, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, producer.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		molaproducer_log.Debugf("<%s> parent of block <%s> is not exist", producer.groupId, block.BlockId)
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetDbMgr().GetBlock(block.PrevBlockId, false, producer.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := IsBlockValid(block, parentBlock)
	if !valid {
		molauser_log.Debugf("<%s> remove invalid block <%s> from cache", producer.groupId, block.BlockId)
		molauser_log.Warningf("<%s> invalid block <%s>", producer.groupId, err.Error())
		return nodectx.GetDbMgr().RmBlock(block.BlockId, true, producer.nodename)
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetDbMgr().GatherBlocksFromCache(block, true, producer.nodename)
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
		molaproducer_log.Debugf("<%s> move block <%s> from cache to chain", producer.groupId, block.BlockId)
		err := nodectx.GetDbMgr().AddBlock(block, false, producer.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetDbMgr().RmBlock(block.BlockId, true, producer.nodename)
		if err != nil {
			return err
		}
	}

	for _, block := range blocks {
		err := nodectx.GetDbMgr().AddProducedBlockCount(producer.groupId, block.ProducerPubKey, producer.nodename)
		if err != nil {
			return err
		}
	}

	molaproducer_log.Debugf("<%s> chain height before recal: <%d>", producer.groupId, producer.grpItem.HighestHeight)
	topBlock, err := nodectx.GetDbMgr().GetBlock(producer.grpItem.HighestBlockId, false, producer.nodename)
	if err != nil {
		return err
	}
	newHeight, newHighestBlockId, err := RecalChainHeight(blocks, producer.grpItem.HighestHeight, topBlock, producer.nodename)
	if err != nil {
		return err
	}
	molaproducer_log.Debugf("<%s> new height <%d>, new highest blockId %v", producer.groupId, newHeight, newHighestBlockId)

	return producer.cIface.UpdChainInfo(newHeight, newHighestBlockId)
}

func (producer *MolassesProducer) applyTrxs(trxs []*quorumpb.Trx) error {
	molaproducer_log.Debugf("<%s> applyTrxs called", producer.groupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetDbMgr().IsTrxExist(trx.TrxId, producer.nodename)
		if err != nil {
			molaproducer_log.Debugf("<%s> %s", producer.groupId, err.Error())
			continue
		}

		if isExist {
			molaproducer_log.Debugf("<%s> trx <%s> existed, update trx", producer.groupId, trx.TrxId)
			nodectx.GetDbMgr().AddTrx(trx)
			continue
		}

		originalData := trx.Data

		if trx.Type == quorumpb.TrxType_POST && producer.grpItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			//just try decrypt it, if failed, save the original encrypted data
			//the reason for that is, for private group, before owner add producer, owner is the only producer,
			//since owner also needs to show POST data, and all announced user will encrypt for owner pubkey
			//owner can actually decrypt POST
			//for other producer, they can not decrpyt POST
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(producer.grpItem.GroupId, trx.Data)
			if err == nil {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}
		} else {
			//decode trx data
			ciperKey, err := hex.DecodeString(producer.grpItem.CipherKey)
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

		molaproducer_log.Debugf("<%s> apply trx <%s>", producer.groupId, trx.TrxId)
		//apply trx content
		switch trx.Type {
		case quorumpb.TrxType_POST:
			molaproducer_log.Debugf("<%s> apply POST trx", producer.groupId)
			nodectx.GetDbMgr().AddPost(trx, producer.nodename)
		case quorumpb.TrxType_AUTH:
			molaproducer_log.Debugf("<%s> apply AUTH trx", producer.groupId)
			nodectx.GetDbMgr().UpdateBlkListItem(trx, producer.nodename)
		case quorumpb.TrxType_PRODUCER:
			molaproducer_log.Debugf("<%s> apply PRODUCER trx", producer.groupId)
			nodectx.GetDbMgr().UpdateProducer(trx, producer.nodename)
			producer.cIface.UpdProducerList()
			producer.cIface.CreateConsensus()
		case quorumpb.TrxType_USER:
			molaproducer_log.Debugf("<%s> apply USER trx", producer.groupId)
			nodectx.GetDbMgr().UpdateUser(trx, producer.nodename)
			producer.cIface.UpdUserList()
		case quorumpb.TrxType_ANNOUNCE:
			molaproducer_log.Debugf("<%s> apply ANNOUNCE trx", producer.groupId)
			nodectx.GetDbMgr().UpdateAnnounce(trx, producer.nodename)
		case quorumpb.TrxType_GROUP_CONFIG:
			molauser_log.Debugf("<%s> apply GROUP_CONFIG trx", producer.groupId)
			nodectx.GetDbMgr().UpdateGroupConfig(trx, producer.nodename)
		case quorumpb.TrxType_SCHEMA:
			molaproducer_log.Debugf("<%s> apply SCHEMA trx", producer.groupId)
			nodectx.GetDbMgr().UpdateSchema(trx, producer.nodename)
		default:
			molaproducer_log.Warningf("<%s> unsupported msgType <%s>", producer.groupId, trx.Type)
		}

		//set trx data to original (encrypted)
		trx.Data = originalData

		//save trx to db
		nodectx.GetDbMgr().AddTrx(trx, producer.nodename)
	}

	return nil
}

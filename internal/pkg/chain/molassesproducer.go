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
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
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
	producer.grp = grp
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

	topBlock, err := GetDbMgr().GetBlock(highestBlockIdWinner, false, producer.NodeName)
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
	chain_log.Infof("%s merge timer ticker...%s", producer.NodeName, t.UTC().String())

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

	chain_log.Debugf("Candidate block decided, block Id : %s", candidateBlkid)
	err := producer.grp.ChainCtx.ProducerAddBlock(producer.blockPool[candidateBlkid])

	if err != nil {
		chain_log.Errorf("save block %s error %s", candidateBlkid, err)
	} else {
		chain_log.Debugf("saved, HighestHeight: %d Highestblkid: %s", producer.grp.Item.HighestHeight, producer.grp.Item.HighestBlockId)
		chain_log.Infof("Send new block out")
		err := producer.trxMgr[producer.grp.ChainCtx.userChannelId].SendBlock(producer.blockPool[candidateBlkid])
		if err != nil {
			chain_log.Warnf(err.Error())
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
	isBlocked, _ := GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey)

	if isBlocked {
		chain_log.Warning("user is blocked by group owner")
		err := errors.New("user auth failed")
		return err
	}

	subBlocks, err := GetDbMgr().GetSubBlock(reqBlockItem.BlockId, producer.NodeName)

	if err != nil {
		return err
	}

	if len(subBlocks) != 0 {
		for _, block := range subBlocks {
			chain_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
			err := producer.trxMgr[producer.grp.ChainCtx.producerChannelId].SendReqBlockResp(&reqBlockItem, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
			if err != nil {
				chain_log.Warnf(err.Error())
			}
		}
		return nil
	} else {
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		//set producer pubkey of empty block
		emptyBlock.BlockId = guuid.New().String()
		emptyBlock.ProducerPubKey = producer.grp.Item.UserSignPubkey
		chain_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)")
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
	isBlocked, _ := GetDbMgr().IsUserBlocked(trx.GroupId, trx.SenderPubkey)

	if isBlocked {
		molaproducer_log.Warning("user is blocked by group owner")
		err := errors.New("user auth failed")
		return err
	}

	isExist, err := GetDbMgr().IsBlockExist(reqBlockItem.BlockId, false, producer.NodeName)
	if err != nil {
		return err
	} else if !isExist {
		return errors.New("Block not exist")
	}

	block, err := GetDbMgr().GetBlock(reqBlockItem.BlockId, false, producer.NodeName)
	if err != nil {
		return err
	}

	isParentExit, err := GetDbMgr().IsParentExist(block.PrevBlockId, false, producer.NodeName)
	if err != nil {
		return err
	}

	if isParentExit {
		molaproducer_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
		parentBlock, err := GetDbMgr().GetParentBlock(reqBlockItem.BlockId, producer.NodeName)
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

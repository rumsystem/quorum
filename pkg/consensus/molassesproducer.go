package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"

	//p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
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
	syncConnTimerPool map[string]*time.Timer
	status            ProducerStatus
	ProduceTimer      *time.Timer
	ProduceDone       chan bool
	statusmu          sync.RWMutex
	addTrxmu          sync.RWMutex
	nodename          string
	cIface            def.ChainMolassesIface
	groupId           string
}

func (producer *MolassesProducer) Init(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molaproducer_log.Debug("Init called")
	producer.grpItem = item
	producer.cIface = iface
	producer.blockPool = make(map[string]*quorumpb.Block)
	//producer.trxPool = make(map[string]*quorumpb.Trx)
	producer.syncConnTimerPool = make(map[string]*time.Timer)
	producer.status = StatusIdle
	producer.nodename = nodename
	producer.groupId = item.GroupId

	molaproducer_log.Infof("<%s> producer created", producer.groupId)
}

// Add trx to trx pool
func (producer *MolassesProducer) AddTrx(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> AddTrx called", producer.groupId)

	producer.addTrxmu.Lock()
	defer producer.addTrxmu.Unlock()

	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, producer.nodename)
	if err != nil {
		return
	}

	if !isAllow {
		molaproducer_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", producer.groupId, trx.SenderPubkey, trx.Type.String())
		return
	}

	//check if trx with same nonce exist, !!Only applied to client which support nonce
	isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.TrxId, trx.Nonce, producer.nodename)
	if isExist {
		molaproducer_log.Debugf("<%s> Trx <%s> with nonce <%d> already packaged, ignore <%s>", producer.groupId, trx.TrxId, trx.Nonce)
		return
	}

	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	producer.trxPool.Store(trx.TrxId, trx)

	molaproducer_log.Debugf("***********_trx_pool_****************")
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
	topBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(producer.grpItem.HighestBlockId, false, producer.nodename)
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
	ks := localcrypto.GetKeystore()
	newBlock, err := rumchaindata.CreateBlockByEthKey(topBlock, trxs, producer.grpItem.UserSignPubkey, ks, "", producer.nodename)
	if err != nil {
		molaproducer_log.Errorf("<%s> create block error", producer.groupId)
		molaproducer_log.Errorf(err.Error())
		return
	}

	//CREATE AND BROADCAST NEW BLOCK BY USING BLOCK_PRODUCED MSG ON PRODUCER CHANNEL
	molaproducer_log.Debugf("<%s> broadcast produced block", producer.groupId)
	connMgr, err := conn.GetConn().GetConnMgr(producer.groupId)
	if err != nil {
		return
	}
	trx, err := producer.cIface.GetTrxFactory().GetBlockProducedTrx("", newBlock)

	connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
	molaproducer_log.Debugf("<%s> produce done, wait for merge", producer.groupId)

	return

}

func (producer *MolassesProducer) addBlockToPool(block *quorumpb.Block) {
	molaproducer_log.Debugf("<%s> addBlockToPool called", producer.groupId)
	/*
		if producer.cIface.IsSyncerReady() {
			return
		}
	*/
	producer.blockPool[block.BlockId] = block
}

func (producer *MolassesProducer) AddProducedBlock(trx *quorumpb.Trx) error {
	molaproducer_log.Debugf("<%s> AddProducedBlock called", producer.groupId)
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
	producer.addBlockToPool(block)

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
	err := producer.cIface.AddBlock(producer.blockPool[candidateBlkid])
	if err != nil {
		molaproducer_log.Errorf("<%s> save block <%s> error <%s>", producer.groupId, candidateBlkid, err)
		if err.Error() == "PARENT_NOT_EXIST" {
			molaproducer_log.Debugf("<%s> parent not found, sync backward for missing blocks from <%s>", producer.groupId, candidateBlkid, err)
			//TOFIX: backward sync, add backward task go gsyncer
			//return producer.cIface.GetChainSyncIface().SyncBackward(candidateBlkid, producer.nodename)
			return nil
		}
	} else {
		molaproducer_log.Debugf("<%s> block saved", producer.groupId)
		//check if I am the winner
		if producer.blockPool[candidateBlkid].ProducerPubKey == producer.grpItem.UserSignPubkey {
			molaproducer_log.Debugf("<%s> winner send new block out", producer.groupId)

			connMgr, err := conn.GetConn().GetConnMgr(producer.groupId)
			if err != nil {
				return err
			}
			err = connMgr.SendBlockPsconn(producer.blockPool[candidateBlkid], conn.UserChannel)
			if err != nil {
				molaproducer_log.Warnf("<%s> <%s>", producer.groupId, err.Error())
			}
		}
	}

	molaproducer_log.Debugf("<%s> merge done", producer.groupId)
	producer.blockPool = make(map[string]*quorumpb.Block)

	return nil
}

package consensus

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var ptbft_log = logging.Logger("ptbft")

var DEFAULT_TRX_PROPOSE_PULSE = 1 * 1000 // 1s

var EMPTY_TRX_BUNDLE = "EMPTY_TRX_BUNDLE"

type PTTask struct {
	Epoch          uint64
	ProposedData   []byte
	acsInsts       *PTAcs
	chAcsDone      chan *PTAcsResult
	taskCtx        context.Context
	taskCancelFunc context.CancelFunc
}

type PTAcsResult struct {
	epoch  uint64
	result map[string][]byte
}

type PTBft struct {
	Config

	currTask *PTTask
	txBuffer *TrxBuffer

	localCtx        context.Context
	localCancelFunc context.CancelFunc

	cIface def.ChainMolassesIface
}

func NewPTBft(ctx context.Context, cfg Config, iface def.ChainMolassesIface) *PTBft {
	ptbft_log.Debugf("<%s> NewPTBft called", cfg.GroupId)
	localCtx, localCancelFunc := context.WithCancel(ctx)
	return &PTBft{
		Config:          cfg,
		txBuffer:        NewTrxBuffer(cfg.GroupId),
		currTask:        nil,
		localCtx:        localCtx,
		localCancelFunc: localCancelFunc,
		cIface:          iface,
	}
}

func (bft *PTBft) AddTrx(tx *quorumpb.Trx) error {
	ptbft_log.Debugf("<%s> AddTrx called, TrxId <%s>", bft.GroupId, tx.TrxId)
	bft.txBuffer.Push(tx)
	return nil
}

// get goroutine id
func goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

func (bft *PTBft) Start() {
	go func() {
		ptbft_log.Debugf("<%s> Start called, id <%d>", bft.GroupId, goid())
		//load propose pulse from config
		interval, err := nodectx.GetNodeCtx().GetChainStorage().GetProducerConsensusConfInterval(bft.GroupId, bft.NodeName)
		if err != nil {
			ptbft_log.Debugf("<%s> GetProducerConsensusConfInterval failed with error <%s>", bft.GroupId, err.Error())
			return
		}

		for {
			ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-bft.localCtx.Done():
					ptbft_log.Debugf("<%s> local ctx finished called, die peaceful", bft.GroupId)
					return
				case <-ticker.C:
					ptbft_log.Debugf("<%s> ticker called at <%d>, propose", bft.GroupId, time.Now().Nanosecond())
					bft.Propose()
				}
			}
		}
	}()
}

func (bft *PTBft) Stop() {
	ptbft_log.Debugf("<%s> Stop called", bft.GroupId)
	if bft.localCancelFunc != nil {
		bft.localCancelFunc()
	}
}

func (bft *PTBft) Propose() {
	ptbft_log.Debugf("<%s> Propose called", bft.GroupId)
	//stop current task if any

	go func() {
		if bft.currTask != nil {
			bft.currTask.taskCancelFunc()
		}
		//select some trxs from buffer
		trxs, err := bft.txBuffer.GetNRandTrx(bft.BatchSize)
		if err != nil {
			ptbft_log.Debugf("<%s> GetNRandTrx failed with error <%s>", bft.GroupId, err.Error())
			return
		}

		trxBundle := &quorumpb.HBTrxBundle{}
		trxBundle.Trxs = append(trxBundle.Trxs, trxs...)

		datab, err := proto.Marshal(trxBundle)
		if err != nil {
			ptbft_log.Debugf("<%s> Marshal failed with error <%s>", bft.GroupId, err.Error())
			return
		}

		if len(datab) == 0 {
			datab = []byte(EMPTY_TRX_BUNDLE)
		}

		currEpoch := bft.cIface.GetCurrEpoch()
		proposedEpoch := currEpoch + 1

		chAcsDone := make(chan *PTAcsResult, 1)
		ctx, cancel := context.WithCancel(bft.localCtx)
		task := &PTTask{
			Epoch:          proposedEpoch,
			ProposedData:   datab,
			acsInsts:       NewPTACS(bft.Config, proposedEpoch, chAcsDone),
			chAcsDone:      chAcsDone,
			taskCtx:        ctx,
			taskCancelFunc: cancel,
		}

		bft.currTask = task
		bft.currTask.acsInsts.InputValue(task.ProposedData)

		//wait till acs done or timeout
		for {
			select {
			case <-bft.currTask.taskCtx.Done():
				ptbft_log.Debugf("<%s> taskCtx done, die peaceful", bft.GroupId)
				return
			case result := <-bft.currTask.chAcsDone:
				ptbft_log.Debugf("<%s> acs done, epoch <%d>, handle result", bft.GroupId, result.epoch)
				bft.acsDone(result)
			}

		}
	}()
}

func (bft *PTBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	//ptbft_log.Debugf("<%s> HandleMessage called, Epoch <%d>", bft.groupId, hbmsg.Epoch)
	if bft.currTask != nil {
		return bft.currTask.acsInsts.HandleHBMessage(hbmsg)
	}

	return nil
}

func (bft *PTBft) acsDone(result *PTAcsResult) {
	ptbft_log.Debugf("<%s> AcsDone called, Epoch <%d>", bft.GroupId, result.epoch)
	trxs := make(map[string]*quorumpb.Trx) //trx_id

	//decode trxs
	for key, value := range result.result {
		//check if result empty
		if string(value) == EMPTY_TRX_BUNDLE {
			continue
		}

		trxBundle := &quorumpb.HBTrxBundle{}
		err := proto.Unmarshal(value, trxBundle)
		if err != nil {
			ptbft_log.Warningf("decode trxs failed for rbc inst %s, err %s", key, err.Error())
			return
		}

		for _, trx := range trxBundle.Trxs {
			if _, ok := trxs[trx.TrxId]; !ok {
				trxs[trx.TrxId] = trx
			}
		}
	}

	//try package trxs with a new block
	if len(trxs) != 0 {
		//Try build block and broadcast it
		err := bft.buildBlock(result.epoch, trxs)
		if err != nil {
			ptbft_log.Warnf("<%s> Build block failed at epoch %d, error %s", bft.GroupId, result.epoch, err.Error())
			return
		}
		//remove outputed trxs from buffer
		for trxId := range trxs {
			err := bft.txBuffer.Delete(trxId)
			ptbft_log.Debugf("<%s> remove packaged trx <%s>", bft.GroupId, trxId)
			if err != nil {
				ptbft_log.Warnf(err.Error())
			}
		}
		//update local BlockId
		bft.cIface.IncCurrBlockId()
	}

	//update and save local epoch
	bft.cIface.IncCurrEpoch()
	bft.cIface.SetLastUpdate(time.Now().UnixNano())
	bft.cIface.SaveChainInfoToDb()
	ptbft_log.Debugf("<%s> ChainInfo updated", bft.GroupId)
}

func (bft *PTBft) buildBlock(epoch uint64, trxs map[string]*quorumpb.Trx) error {
	ptbft_log.Debugf("<%s> buildBlock called, epoch <%d>", bft.GroupId, epoch)
	//try build block by using trxs

	ptbft_log.Debugf("<%s> sort trx", bft.GroupId)
	trxToPackage := bft.sortTrx(trxs)

	currBlockId := bft.cIface.GetCurrBlockId()
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(bft.GroupId, currBlockId, false, bft.NodeName)

	if err != nil {
		ptbft_log.Debugf("<%s> get block parent failed, <%s>", bft.GroupId, err.Error())
		return err
	} else {
		ptbft_log.Debugf("<%s> start build block with parent <%d> ", bft.GroupId, parent.BlockId)
		ks := localcrypto.GetKeystore()

		newBlock, err := rumchaindata.CreateBlockByEthKey(parent, epoch, trxToPackage, bft.MyPubkey, ks, "", bft.NodeName)

		if err != nil {
			ptbft_log.Debugf("<%s> build block failed <%s>", bft.GroupId, err.Error())
			return err
		}

		//save it
		//ptbft_log.Debugf("<%s> save block just built to local db", bft.producer.groupId)
		err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(newBlock, false, bft.NodeName)
		if err != nil {
			return err
		}

		//apply trxs
		if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
			bft.cIface.ApplyTrxsProducerNode(trxToPackage, bft.NodeName)
		} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
			bft.cIface.ApplyTrxsFullNode(trxToPackage, bft.NodeName)
		}

		//broadcast it
		ptbft_log.Debugf("<%s> broadcast block just built to user channel", bft.GroupId)
		connMgr, err := conn.GetConn().GetConnMgr(bft.GroupId)
		if err != nil {
			return err
		}
		err = connMgr.BroadcastBlock(newBlock)
		if err != nil {
			ptbft_log.Debugf("<%s> Broadcast failed <%s>", bft.GroupId, err.Error())
		}
	}

	return nil
}

// sort trxs by using timestamp
type TrxSlice []*quorumpb.Trx

func (a TrxSlice) Len() int {
	return len(a)
}
func (a TrxSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a TrxSlice) Less(i, j int) bool {
	return a[j].TimeStamp < a[i].TimeStamp
}

func (bft *PTBft) sortTrx(trxs map[string]*quorumpb.Trx) []*quorumpb.Trx {
	result := []*quorumpb.Trx{}
	container := make(map[string][]*quorumpb.Trx)

	//group trxs by using sender Pubkey (group trxs from same sender)
	for _, trx := range trxs {
		container[trx.SenderPubkey] = append(container[trx.SenderPubkey], trx)
	}

	//sort each grouped trxs by using timestamp (from small to large)
	for _, trxs := range container {
		sort.Sort(sort.Reverse(TrxSlice(trxs)))
	}

	var senderKeys []string
	//get all key (sender pubkey) from container
	for key, _ := range container {
		senderKeys = append(senderKeys, key)
	}

	//sort sender key
	sort.Strings(senderKeys)

	for _, key := range senderKeys {
		//skip owner trxs
		if key == bft.OwnerPubKey {
			continue
		}
		//append
		result = append(result, container[key]...)
	}

	//append any trxs from owner at the end of trxs slice
	if ownertrxs, ok := container[bft.OwnerPubKey]; ok {
		result = append(result, ownertrxs...)
	}

	return result
}

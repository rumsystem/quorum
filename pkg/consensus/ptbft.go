package consensus

import (
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var ptbft_log = logging.Logger("ptbft")

var DEFAULT_TRX_PROPOSE_PULSE = 5 * 1000 // 1s
var EMPTY_TRX_BUNDLE = "EMPTY_TRX_BUNDLE"

type PTTask struct {
	Epoch        uint64
	ProposedData []byte
	acsInsts     *PTAcs
}

type PTBft struct {
	Config
	groupId  string
	producer *MolassesProducer
	CurrTask *PTTask
	txBuffer *TrxBuffer
	status   BftStatus

	ticker     *time.Ticker
	tickerdone chan bool
	taskdone   chan bool
}

func NewPTBft(cfg Config, producer *MolassesProducer) *PTBft {
	ptbft_log.Debugf("<%s> NewPTBft called", producer.groupId)
	return &PTBft{
		Config:     cfg,
		groupId:    producer.groupId,
		producer:   producer,
		txBuffer:   NewTrxBuffer(producer.groupId),
		CurrTask:   nil,
		status:     IDLE,
		ticker:     nil,
		tickerdone: make(chan bool),
		taskdone:   make(chan bool),
	}
}

func (bft *PTBft) AddTrx(tx *quorumpb.Trx) error {
	ptbft_log.Debugf("<%s> AddTrx called, TrxId <%s>", bft.groupId, tx.TrxId)
	bft.txBuffer.Push(tx)
	return nil
}

func (bft *PTBft) Start() {
	ptbft_log.Debugf("<%s> Start called", bft.groupId)
	go func() {
		bft.ticker = time.NewTicker(time.Duration(DEFAULT_TRX_PROPOSE_PULSE) * time.Millisecond)
		bft.status = RUNNING
		for {
			select {
			case <-bft.tickerdone:
				ptbft_log.Debugf("<%s> TickerDone called", bft.groupId)
				return
			case <-bft.ticker.C:
				ptbft_log.Debugf("<%s> ticker called at <%d>", bft.groupId, time.Now().Nanosecond())
				bft.Propose()
			}
		}
		bft.ticker.Stop()
	}()
}

func (bft *PTBft) Stop() {
	ptbft_log.Debugf("<%s> Stop called", bft.groupId)
	if bft.status != RUNNING {
		ptbft_log.Debugf("<%s> BFT not RUNNING, can not stop propose", bft.groupId)
		return
	}

	bft.status = CLOSED

	//stop current task
	bft.taskdone <- true

	bft.ticker.Stop()
	bft.tickerdone <- true

	ptbft_log.Debugf("<%s> StopPropose done", bft.groupId)
}

func (bft *PTBft) Propose() error {
	ptbft_log.Debugf("<%s> NewProposeTask called", bft.groupId)

	//select some trxs from buffer
	trxs, err := bft.txBuffer.GetNRandTrx(bft.BatchSize)
	if err != nil {
		return err
	}

	trxBundle := &quorumpb.HBTrxBundle{}
	trxBundle.Trxs = append(trxBundle.Trxs, trxs...)

	datab, err := proto.Marshal(trxBundle)
	if err != nil {
		return err
	}

	if len(datab) == 0 {
		datab = []byte(EMPTY_TRX_BUNDLE)
	}

	currEpoch := bft.producer.cIface.GetCurrEpoch()
	proposedEpoch := currEpoch + 1

	task := &PTTask{
		Epoch:        proposedEpoch,
		ProposedData: datab,
		acsInsts:     NewPTAcs(bft.Config, bft, proposedEpoch),
	}

	bft.CurrTask = task

	//run task
	go func() {
		ptbft_log.Debugf("<%s> task <%d> start", bft.groupId, bft.CurrTask.Epoch)
		bft.CurrTask.acsInsts.InputValue(task.ProposedData)
	}()

	//wait here till get task done signal
	<-bft.taskdone

	ptbft_log.Debugf("<%s> task <%d> done", bft.groupId, task.Epoch)
	return nil
}

func (bft *PTBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	ptbft_log.Debugf("<%s> HandleMessage called, Epoch <%d>", bft.groupId, hbmsg.Epoch)

	if bft.CurrTask != nil {
		return bft.CurrTask.acsInsts.HandleHBMessage(hbmsg)
	}

	return nil
}

func (bft *PTBft) AcsDone(epoch uint64, result map[string][]byte) {
	ptbft_log.Debugf("<%s> AcsDone called, Epoch <%d>", bft.producer.groupId, epoch)
	trxs := make(map[string]*quorumpb.Trx) //trx_id

	//decode trxs
	for key, value := range result {
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
		err := bft.buildBlock(epoch, trxs)
		if err != nil {
			ptbft_log.Warnf("<%s> Build block failed at epoch %d, error %s", bft.producer.groupId, epoch, err.Error())
			return
		}
		//remove outputed trxs from buffer
		for trxId := range trxs {
			err := bft.txBuffer.Delete(trxId)
			ptbft_log.Debugf("<%s> remove packaged trx <%s>", bft.producer.groupId, trxId)
			if err != nil {
				ptbft_log.Warnf(err.Error())
			}
		}
		//update local BlockId
		bft.producer.cIface.IncCurrBlockId()
	}

	//update and save local epoch
	bft.producer.cIface.IncCurrEpoch()
	bft.producer.cIface.SetLastUpdate(time.Now().UnixNano())
	bft.producer.cIface.SaveChainInfoToDb()
	ptbft_log.Debugf("<%s> ChainInfo updated", bft.producer.groupId)

	bft.taskdone <- true
}

func (bft *PTBft) buildBlock(epoch uint64, trxs map[string]*quorumpb.Trx) error {
	ptbft_log.Debugf("<%s> buildBlock called, epoch <%d>", bft.producer.groupId, epoch)
	//try build block by using trxs

	ptbft_log.Debugf("<%s> sort trx", bft.producer.groupId)
	trxToPackage := bft.sortTrx(trxs)

	currBlockId := bft.producer.cIface.GetCurrBlockId()
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(bft.producer.groupId, currBlockId, false, bft.producer.nodename)

	if err != nil {
		ptbft_log.Debugf("<%s> get block parent failed, <%s>", bft.producer.groupId, err.Error())
		return err
	} else {
		ptbft_log.Debugf("<%s> start build block with parent <%d> ", bft.producer.groupId, parent.BlockId)
		ks := localcrypto.GetKeystore()

		newBlock, err := rumchaindata.CreateBlockByEthKey(parent, epoch, trxToPackage, bft.producer.grpItem.UserSignPubkey, ks, "", bft.producer.nodename)

		if err != nil {
			ptbft_log.Debugf("<%s> build block failed <%s>", bft.producer.groupId, err.Error())
			return err
		}

		//save it
		ptbft_log.Debugf("<%s> save block just built to local db", bft.producer.groupId)
		err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(newBlock, false, bft.producer.nodename)
		if err != nil {
			return err
		}

		//apply trxs
		if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
			bft.producer.cIface.ApplyTrxsProducerNode(trxToPackage, bft.producer.nodename)
		} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
			bft.producer.cIface.ApplyTrxsFullNode(trxToPackage, bft.producer.nodename)
		}

		//broadcast it
		ptbft_log.Debugf("<%s> broadcast block just built to user channel", bft.producer.groupId)
		connMgr, err := conn.GetConn().GetConnMgr(bft.producer.groupId)
		if err != nil {
			return err
		}
		err = connMgr.BroadcastBlock(newBlock)
		if err != nil {
			ptbft_log.Debugf("<%s> Broadcast failed <%s>", bft.producer.groupId, err.Error())
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
		if key == bft.producer.grpItem.OwnerPubKey {
			continue
		}
		//append
		result = append(result, container[key]...)
	}

	//append any trxs from owner at the end of trxs slice
	if ownertrxs, ok := container[bft.producer.grpItem.OwnerPubKey]; ok {
		result = append(result, ownertrxs...)
	}

	return result
}

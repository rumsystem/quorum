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

var trx_bft_log = logging.Logger("tbft")

var DEFAULT_PROPOSE_PULSE = 1 * 1000       // 1s
var MAXIMUM_TRX_BUNDLE_LENGTH = 900 * 1024 //900Kib

type ProposeTask struct {
	Epoch          uint64
	ProposedData   []byte
	DelayStartTime int
}

type ProposeStatus uint

const (
	IDLE ProposeStatus = iota
	RUNNING
	CLOSED
)

type TrxBft struct {
	Config
	groupId  string
	producer *MolassesProducer
	CurrTask *ProposeTask
	acsInsts *TrxACS
	txBuffer *TrxBuffer
	taskq    chan *ProposeTask

	taskdone   chan struct{}
	stopnotify chan struct{}

	status ProposeStatus
}

func NewTrxBft(cfg Config, producer *MolassesProducer) *TrxBft {
	trx_bft_log.Debugf("<%s> NewTrxBft called", producer.groupId)
	return &TrxBft{
		Config:     cfg,
		groupId:    producer.groupId,
		producer:   producer,
		txBuffer:   NewTrxBuffer(producer.groupId),
		taskq:      make(chan *ProposeTask),
		taskdone:   make(chan struct{}),
		stopnotify: make(chan struct{}),
		status:     IDLE,
	}
}

func (bft *TrxBft) StartPropose() {
	trx_bft_log.Debugf("<%s> StartPropose called", bft.groupId)

	//start taskq
	go func() {
		for task := range bft.taskq {
			bft.runTask(task)
		}
		bft.stopnotify <- struct{}{}
	}()

	//add first task
	task, _ := bft.NewProposeTask()
	bft.addTask(task)
}

func (bft *TrxBft) KillAndRunNextRound() {
	trx_bft_log.Debugf("<%s> KillCurrentTask called", bft.groupId)

	//finish current task
	bft.taskdone <- struct{}{}

	//bft.CurrTask = nil
	//bft.acsInsts = nil

	task, _ := bft.NewProposeTask()
	bft.addTask(task)
}

func (bft *TrxBft) addTask(task *ProposeTask) {
	trx_bft_log.Debugf("<%s> bft addTask called", bft.groupId)
	go func() {
		if bft.status != CLOSED {
			bft.taskq <- task
		}
	}()
}

func (bft *TrxBft) runTask(task *ProposeTask) error {
	trx_bft_log.Debugf("<%s> runTask called, epoch <%d>", bft.groupId, task.Epoch)
	go func() {
		//create new acs and try propose something
		trx_bft_log.Debugf("<%s> wait <%d> ms", bft.groupId, task.DelayStartTime)
		time.Sleep(time.Duration(task.DelayStartTime) * time.Millisecond)

		bft.CurrTask = task
		bft.acsInsts = NewTrxACS(bft.Config, bft, task.Epoch)
		bft.acsInsts.InputValue(task.ProposedData)
	}()

	//wait here
	<-bft.taskdone
	return nil
}

func (bft *TrxBft) NewProposeTask() (*ProposeTask, error) {
	trx_bft_log.Debugf("<%s> NewProposeTask called", bft.groupId)

	//select some trxs from buffer
	trxs, err := bft.txBuffer.GetNRandTrx(bft.BatchSize)
	if err != nil {
		return nil, err
	}

	//list all trxs
	trx_bft_log.Debugf("<%s> trxs to propose", bft.groupId)
	for _, trx := range trxs {
		trx_bft_log.Debugf("<%s> trx <%s>", bft.groupId, trx.TrxId)
	}

	var datab []byte
	for {
		trxBundle := &quorumpb.HBTrxBundle{}
		trxBundle.Trxs = append(trxBundle.Trxs, trxs...)

		datab, err = proto.Marshal(trxBundle)
		if err != nil {
			return nil, err
		}

		if len(datab) == 0 {
			datab = []byte("EMPTY")
			break
		} else if len(datab) <= MAXIMUM_TRX_BUNDLE_LENGTH {
			break
		}

		//remove last trxs from the slice and try again
		trxs = trxs[:len(trxs)-1]
	}

	currEpoch := bft.producer.cIface.GetCurrEpoch()
	proposedEpoch := currEpoch + 1

	task := &ProposeTask{
		Epoch:          proposedEpoch,
		ProposedData:   datab,
		DelayStartTime: DEFAULT_PROPOSE_PULSE,
	}

	return task, nil
}

func (bft *TrxBft) StopPropose() {
	trx_bft_log.Debugf("<%s> StopPropose called", bft.groupId)
	bft.status = CLOSED
	safeCloseTaskQ(bft.taskq)
	safeClose(bft.taskdone)
	if bft.stopnotify != nil {
		signcount := 1
		for range bft.stopnotify {
			signcount++
			//wait stop sign and set idle
			if signcount == 1 { // taskq
				close(bft.stopnotify)
				trx_bft_log.Debugf("<%s> bft stop propose done.")
			}
		}
	}
}

func safeClose(ch chan struct{}) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}

func safeCloseTaskQ(ch chan *ProposeTask) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}

func (bft *TrxBft) AddTrx(tx *quorumpb.Trx) error {
	trx_bft_log.Debugf("<%s> AddTrx called, TrxId <%s>", bft.groupId, tx.TrxId)
	bft.txBuffer.Push(tx)

	//for debug only added by cuicat
	//list all trxs in buffer
	trxs, err := bft.txBuffer.GetAllTrxInBuffer()
	if err != nil {
		return err
	}

	trx_bft_log.Debugf("<%s> all trx in buffer", bft.groupId)
	for _, trx := range trxs {
		trx_bft_log.Debugf("<%s> TrxId <%s>", bft.groupId, trx.TrxId)
	}

	return nil
}

func (bft *TrxBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	trx_bft_log.Debugf("<%s> HandleMessage called, Epoch <%d>", bft.groupId, hbmsg.Epoch)

	if bft.acsInsts != nil && hbmsg.Epoch < bft.acsInsts.Epoch {
		trx_bft_log.Warnf("message from old epoch, ignore")
		return nil
	}

	//handle msg
	return bft.acsInsts.HandleMessage(hbmsg)
}

func (bft *TrxBft) AcsDone(epoch uint64, result map[string][]byte) {
	trx_bft_log.Debugf("<%s> AcsDone called, Epoch <%d>", bft.producer.groupId, epoch)
	trxs := make(map[string]*quorumpb.Trx) //trx_id

	//decode trxs
	for key, value := range result {
		//check if result empty
		if string(value) == "EMPTY" {
			continue
		}

		trxBundle := &quorumpb.HBTrxBundle{}
		err := proto.Unmarshal(value, trxBundle)
		if err != nil {
			trx_bft_log.Warningf("decode trxs failed for rbc inst <%s> with error <%s>", key, err.Error())
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
			trx_bft_log.Warnf("<%s> Build block failed at epoch <%d>, error <%s>", bft.producer.groupId, epoch, err.Error())
			return
		}
		//remove packaged trxs from buffer
		for trxId := range trxs {
			err := bft.txBuffer.Delete(trxId)
			trx_bft_log.Debugf("<%s> remove packaged trx <%s>", bft.producer.groupId, trxId)
			if err != nil {
				trx_bft_log.Warnf(err.Error())
			}
		}

		//get all trxs in buffer after delete
		trxs, err := bft.txBuffer.GetAllTrxInBuffer()
		if err != nil {
			trx_bft_log.Warnf(err.Error())
		}

		//list all trxs
		trx_bft_log.Debugf("<%s> after delete, all trx in buffer", bft.producer.groupId)
		for _, trx := range trxs {
			trx_bft_log.Debugf("<%s> TrxId <%s>", bft.producer.groupId, trx.TrxId)
		}

		//update local BlockId
		bft.producer.cIface.IncCurrBlockId()
	}

	//update and save local epoch
	bft.producer.cIface.IncCurrEpoch()
	bft.producer.cIface.SetLastUpdate(time.Now().UnixNano())
	bft.producer.cIface.SaveChainInfoToDb()
	trx_bft_log.Debugf("<%s> ChainInfo updated", bft.producer.groupId)

	//finish current task
	bft.taskdone <- struct{}{}

	task, _ := bft.NewProposeTask()
	bft.addTask(task)
}

func (bft *TrxBft) buildBlock(epoch uint64, trxs map[string]*quorumpb.Trx) error {
	trx_bft_log.Debugf("<%s> buildBlock called, epoch <%d>", bft.producer.groupId, epoch)
	//try build block by using trxs
	sortedTrxs := bft.sortTrx(trxs)
	trx_bft_log.Debugf("<%s> sorted trxs", bft.producer.groupId)
	//list all sorted trx
	for _, trx := range sortedTrxs {
		trx_bft_log.Debugf("<%s> TrxId <%s>", bft.producer.groupId, trx.TrxId)
	}

	var trxToPackage []*quorumpb.Trx

	//check total trxs size
	totalTrxSizeInBytes := 0
	for _, trx := range sortedTrxs {
		datab, _ := proto.Marshal(trx)
		if totalTrxSizeInBytes+len(datab) <= MAXIMUM_TRX_BUNDLE_LENGTH {
			trxToPackage = append(trxToPackage, trx)
			totalTrxSizeInBytes = totalTrxSizeInBytes + len(datab)
		} else {
			break
		}
	}
	//list all trx to package
	trx_bft_log.Debugf("<%s> trx to package", bft.producer.groupId)
	for _, trx := range trxToPackage {
		trx_bft_log.Debugf("<%s> TrxId <%s>", bft.producer.groupId, trx.TrxId)
	}

	currBlockId := bft.producer.cIface.GetCurrBlockId()
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(bft.producer.groupId, currBlockId, false, bft.producer.nodename)

	if err != nil {
		trx_bft_log.Debugf("<%s> get block parent failed, <%s>", bft.producer.groupId, err.Error())
		return err
	} else {
		trx_bft_log.Debugf("<%s> start build block with parent <%d> ", bft.producer.groupId, parent.BlockId)
		ks := localcrypto.GetKeystore()

		newBlock, err := rumchaindata.CreateBlockByEthKey(parent, epoch, trxToPackage, false, bft.producer.grpItem.UserSignPubkey, ks, "", bft.producer.nodename)

		if err != nil {
			trx_bft_log.Debugf("<%s> build block failed <%s>", bft.producer.groupId, err.Error())
			return err
		}

		//save it
		trx_bft_log.Debugf("<%s> save block just built to local db", bft.producer.groupId)
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
		trx_bft_log.Debugf("<%s> broadcast block just built to user channel", bft.producer.groupId)
		connMgr, err := conn.GetConn().GetConnMgr(bft.producer.groupId)
		if err != nil {
			return err
		}
		err = connMgr.BroadcastBlock(newBlock)
		if err != nil {
			trx_acs_log.Debugf("<%s> Broadcast failed <%s>", bft.producer.groupId, err.Error())
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

func (bft *TrxBft) sortTrx(trxs map[string]*quorumpb.Trx) []*quorumpb.Trx {
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

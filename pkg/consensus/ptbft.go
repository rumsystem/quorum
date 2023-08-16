package consensus

import (
	"context"
	"sort"
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

var EMPTY_TRX_BUNDLE = "EMPTY_TRX_BUNDLE"
var MAXIMUM_TRX_BUNDLE_LENGTH = 900 * 1024 //900Kib

type PTBft struct {
	Config
	cIface            def.ChainMolassesIface
	txBuffer          *TrxBuffer
	chainCtx          context.Context
	currTask          *PTTask
	bftCtx            context.Context
	bftCancelFunc     context.CancelFunc
	CurrConsensusInfo *quorumpb.ConsensusInfo
}

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

func NewPTBft(ctx context.Context, cfg Config, iface def.ChainMolassesIface) *PTBft {
	ptbft_log.Debugf("<%s> NewPTBft called", cfg.GroupId)
	ptbft := &PTBft{
		Config:   cfg,
		cIface:   iface,
		txBuffer: NewTrxBuffer(cfg.GroupId),
		chainCtx: ctx,
		currTask: nil,
	}
	return ptbft
}

func (bft *PTBft) getNextTask() (*PTTask, error) {
	ptbft_log.Debugf("<%s> getNextTask called", bft.GroupId)

	trxs, err := bft.txBuffer.GetNRandTrx(bft.BatchSize)
	if err != nil {
		ptbft_log.Debugf("<%s> GetNRandTrx failed with error <%s>", bft.GroupId, err.Error())
		return nil, err
	}

	var datab []byte
	for {
		trxBundle := &quorumpb.HBTrxBundle{}
		trxBundle.Trxs = append(trxBundle.Trxs, trxs...)

		datab, err = proto.Marshal(trxBundle)
		if err != nil {
			ptbft_log.Debugf("<%s> Marshal failed with error <%s>", bft.GroupId, err.Error())
			return nil, err
		}

		if len(datab) == 0 {
			datab = []byte(EMPTY_TRX_BUNDLE)
			break
		} else if len(datab) <= MAXIMUM_TRX_BUNDLE_LENGTH {
			ptbft_log.Debugf("<%s> datab length <%d> is ok, continue", bft.GroupId, len(datab))
			break
		}

		ptbft_log.Debugf("<%s> datab length <%d> is too long, remove last trx and try again", bft.GroupId, len(datab))
		trxs = trxs[:len(trxs)-1]
	}

	currEpoch := bft.cIface.GetCurrEpoch()
	proposedEpoch := currEpoch + 1

	ptbft_log.Debugf("<%s> >>> Load task: epoch <%d> try propose the following trxs:", bft.GroupId, proposedEpoch)
	for _, trx := range trxs {
		ptbft_log.Debugf("trx : <%s>", trx.TrxId)
	}
	if len(trxs) == 0 {
		ptbft_log.Debugf("<>")
	}

	chAcsDone := make(chan *PTAcsResult, 1)
	ctx, cancel := context.WithCancel(bft.chainCtx)
	task := &PTTask{
		Epoch:          proposedEpoch,
		ProposedData:   datab,
		acsInsts:       NewPTACS(bft.Config, bft.CurrConsensusInfo, proposedEpoch, chAcsDone),
		chAcsDone:      chAcsDone,
		taskCtx:        ctx,
		taskCancelFunc: cancel,
	}

	return task, nil
}

func (bft *PTBft) ProposeWorker() {
	ptbft_log.Debugf("<%s> ProposeWorker called", bft.GroupId)
	interval, err := nodectx.GetNodeCtx().GetChainStorage().GetProducerConsensusConfInterval(bft.GroupId, bft.NodeName)
	if err != nil {
		ptbft_log.Debugf("<%s> GetProducerConsensusConfInterval failed with error <%s>", bft.GroupId, err.Error())
		return
	}

	for {
		select {
		case <-bft.chainCtx.Done():
			ptbft_log.Debugf("<%s> chainCtx done, ProposeWorker exit", bft.GroupId)
			if bft.currTask != nil {
				bft.currTask.taskCancelFunc()
				bft.currTask = nil
			}
			return

		case <-bft.bftCtx.Done():
			ptbft_log.Debugf("<%s> bftCtx done, ProposeWorker exit", bft.GroupId)
			if bft.currTask != nil {
				bft.currTask.taskCancelFunc()
				bft.currTask = nil
			}

			return

		case <-time.After(time.Duration(interval) * time.Millisecond):
			if bft.chainCtx.Err() != nil {
				ptbft_log.Debugf("<%s> chainCtx err, ProposeWorker exit", bft.GroupId)
				return
			}

			if bft.bftCtx.Err() != nil {
				ptbft_log.Debugf("<%s> bftCtx err, ProposeWorker exit", bft.GroupId)
				return
			}

			//cancel previous task
			if bft.currTask != nil {
				bft.currTask.taskCancelFunc()
				bft.currTask = nil
			}

			//get next task
			bftTask, err := bft.getNextTask()
			if err != nil {
				ptbft_log.Debugf("<%s> getNextTask failed with error <%s>", bft.GroupId, err.Error())
				continue
			}

			bft.currTask = bftTask
			go func() {
				bftTask.acsInsts.InputValue(bftTask.ProposedData)
				select {
				case <-bft.currTask.taskCtx.Done():
					ptbft_log.Debugf("<%s> PTBftWorker acs done, epoch <%d>, taskCtx done without result", bft.GroupId, bftTask.Epoch)
					return
				case result := <-bftTask.chAcsDone:
					ptbft_log.Debugf("<%s> PTBftWorker acs done, epoch <%d>, handle result", bft.GroupId, result.epoch)
					bft.acsDone(result)
					return
				}
			}()
		}
	}
}

func (bft *PTBft) AddTrx(tx *quorumpb.Trx) error {
	ptbft_log.Debugf("<%s> AddTrx called, TrxId <%s>", bft.GroupId, tx.TrxId)
	bft.txBuffer.Push(tx)
	return nil
}

func (bft *PTBft) Start() {
	ptbft_log.Debugf("<%s> Start called", bft.GroupId)
	if bft.bftCtx != nil {
		bft.bftCancelFunc()
	}

	bft.bftCtx, bft.bftCancelFunc = context.WithCancel(bft.chainCtx)
	go bft.ProposeWorker()
}

func (bft *PTBft) Stop() {
	ptbft_log.Debugf("<%s> Stop called", bft.GroupId)
	if bft.bftCtx != nil {
		bft.bftCancelFunc()
		bft.bftCtx = nil
	}
}

func (bft *PTBft) HandleHBMessage(hbMsg *quorumpb.HBMsgv1) error {
	//ptbft_log.Debugf("<%s> HandleMessage called, Epoch <%d>", bft.groupId, hbmsg.Epoch)
	if bft.currTask != nil {
		return bft.currTask.acsInsts.HandleHBMessage(hbMsg)
	}
	return nil
}

func (bft *PTBft) acsDone(result *PTAcsResult) {
	//ptbft_log.Debugf("<%s> AcsDone called, Epoch <%d>", bft.GroupId, result.epoch)
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

	ptbft_log.Debugf("<%s> >>> epoch <%d> done, trxs to package", bft.GroupId, result.epoch)
	//try package trxs with a new block
	if len(trxs) != 0 {
		for _, trx := range trxs {
			ptbft_log.Debugf("<%s> --- <%s>", bft.GroupId, trx.TrxId)
		}

		//Try build block and broadcast it
		err := bft.buildBlock(result.epoch, trxs)
		if err != nil {
			ptbft_log.Warnf("<%s> Build block failed at epoch <%d>, error <%s>", bft.GroupId, result.epoch, err.Error())
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
	} else {
		ptbft_log.Debugf("<%s> --- <>", bft.GroupId)
	}

	//update and save local epoch
	bft.cIface.IncCurrEpoch()
	bft.cIface.SetLastUpdate(time.Now().UnixNano())
	bft.cIface.SaveChainInfoToDb()
	//ptbft_log.Debugf("<%s> ChainInfo updated", bft.GroupId)
}

func (bft *PTBft) buildBlock(epoch uint64, trxs map[string]*quorumpb.Trx) error {
	ptbft_log.Debugf("<%s> buildBlock called, epoch <%d>", bft.GroupId, epoch)
	//try build block by using trxs
	//ptbft_log.Debugf("<%s> sort trx", bft.GroupId)
	sortedTrxs := bft.sortTrx(trxs)
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

	ptbft_log.Debugf("<%s> trxs to package, total size in bytes <%d>", bft.GroupId, totalTrxSizeInBytes)
	for _, trx := range trxToPackage {
		ptbft_log.Debugf("---> <%s>", trx.TrxId)
	}

	currBlockId := bft.cIface.GetCurrBlockId()
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(bft.GroupId, currBlockId, false, bft.NodeName)

	if err != nil {
		ptbft_log.Debugf("<%s> get block parent failed, <%s>", bft.GroupId, err.Error())
		return err
	} else {
		ptbft_log.Debugf("<%s> start build block with parent <%d> ", bft.GroupId, parent.BlockId)
		ks := localcrypto.GetKeystore()

		//TBD add consensus info here
		newBlock, err := rumchaindata.CreateBlockByEthKey(parent, nil, trxToPackage, bft.MyPubkey, ks, "", bft.NodeName)
		if err != nil {
			ptbft_log.Debugf("<%s> build block failed <%s>", bft.GroupId, err.Error())
			return err
		}

		//save it
		ptbft_log.Debugf("<%s> save block just built to local db", bft.GroupId)
		err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(newBlock, false, bft.NodeName)
		if err != nil {
			return err
		}

		//apply trxs
		if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
			//bft.cIface.ApplyTrxsProducerNode(trxToPackage, bft.NodeName)
		} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
			//bft.cIface.ApplyTrxsFullNode(trxToPackage, bft.NodeName)
			bft.cIface.ApplyTrxsRumLiteNode(trxToPackage, bft.NodeName)
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

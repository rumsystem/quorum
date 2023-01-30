package consensus

import (
	"sort"
	"sync"
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

type TrxBft struct {
	Config
	producer *MolassesProducer
	acsInsts sync.Map //[int64]*TrxACS //map key is epoch
	txBuffer *TrxBuffer
}

func NewTrxBft(cfg Config, producer *MolassesProducer) *TrxBft {
	trx_bft_log.Debugf("<%s> NewTrxBft called", producer.groupId)
	return &TrxBft{
		Config:   cfg,
		producer: producer,
		txBuffer: NewTrxBuffer(producer.groupId),
	}
}

func (bft *TrxBft) AddTrx(tx *quorumpb.Trx) error {
	trx_bft_log.Debugf("<%s> AddTrx called", bft.producer.groupId)
	bft.txBuffer.Push(tx)

	found := false
	f := func(key, value any) bool {
		TopEpoch := bft.producer.cIface.GetCurrEpoch() + 1 //proposed but not finished epoch is current group epoch + 1 (next epoch)
		if key == TopEpoch {
			found = true
		}
		return true
	}

	bft.acsInsts.Range(f)
	if found {
		trx_bft_log.Debugf("<%s> Trx saved to TrxBuffer, wait to be propose", tx.TrxId)
		return nil
	}

	//try propose with next epoch
	newEpoch := bft.producer.cIface.GetCurrEpoch() + 1
	trx_bft_log.Debugf("Try propose with new Epoch <%d>", newEpoch)
	bft.propose(newEpoch)
	return nil
}

func (bft *TrxBft) AddSudoTrx(tx *quorumpb.Trx) error {
	trx_bft_log.Debugf("AddSudoTrx called")

	//check if sudotrx is from group owner
	if bft.producer.grpItem.OwnerPubKey != tx.SenderPubkey {
		trx_bft_log.Warnf("SudoTrx <%s> from non owner <%s>, ignore", tx.TrxId, tx.SenderPubkey)
		return nil
	}

	//check if I am owner
	if bft.producer.grpItem.OwnerPubKey != bft.producer.grpItem.UserSignPubkey {
		trx_bft_log.Warnf("Ignore by me, owner node will handle sudotrx <%s>", tx.SenderPubkey)
		return nil
	}

	//SudoTrx will bypass consensus, owner node will generate SudoBlock by itself
	return nil
}

func (bft *TrxBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	trx_bft_log.Debugf("<%s> HandleMessage called, Epoch <%d>", bft.producer.groupId, hbmsg.Epoch)
	var acs *TrxACS
	inst, ok := bft.acsInsts.Load(hbmsg.Epoch)

	if !ok {
		if hbmsg.Epoch <= bft.producer.cIface.GetCurrEpoch() {
			trx_bft_log.Warnf("message from old epoch, ignore")
			return nil
		}
		//create newTrxAcs and save it
		acs = NewTrxACS(bft.Config, bft, hbmsg.Epoch)
		//TrxACS will be cast by syncmap to type ANY automatically
		bft.acsInsts.Store(hbmsg.Epoch, acs)
		trx_bft_log.Debugf("Create new ACS %d", hbmsg.Epoch)
	} else {
		//get acs from syncmap, cast from type ANY back to TrxAcs
		acs = inst.(*TrxACS)
	}

	return acs.HandleMessage(hbmsg)
}

func (bft *TrxBft) AcsDone(epoch int64, result map[string][]byte) {
	trx_bft_log.Debugf("<%s> AcsDone called, Epoch <%d>", bft.producer.groupId, epoch)
	trxs := make(map[string]*quorumpb.Trx) //trx_id

	//decode trxs
	for key, value := range result {
		trxBundle := &quorumpb.HBTrxBundle{}
		err := proto.Unmarshal(value, trxBundle)
		if err != nil {
			trx_bft_log.Warningf("decode trxs failed for rbc inst %s, err %s", key, err.Error())
			return
		}

		for _, trx := range trxBundle.Trxs {
			if _, ok := trxs[trx.TrxId]; !ok {
				trxs[trx.TrxId] = trx
			}
		}
	}

	buildBlockDone := false

	//Try build block
	err := bft.buildBlock(epoch, trxs)
	if err != nil {
		trx_bft_log.Warnf("????????????? <%s> Build block failed at epoch %d, error %s", bft.producer.groupId, epoch, err.Error())
	} else {
		buildBlockDone = true
	}

	//clear acs for finished epoch
	if buildBlockDone {
		//remove outputed trxs from buffer
		for trxId := range trxs {
			err := bft.txBuffer.Delete(trxId)
			trx_bft_log.Debugf("<%s> remove packaged trx <%s>", bft.producer.groupId, trxId)
			if err != nil {
				trx_bft_log.Warnf(err.Error())
			}
		}

		//update chain epoch
		bft.producer.cIface.IncCurrEpoch()
		bft.producer.cIface.SetLastUpdate(time.Now().UnixNano())
		bft.producer.cIface.SaveChainInfoToDb()
		trx_bft_log.Debugf("<%s> ChainInfo updated", bft.producer.groupId)

		//nodectx.GetNodeCtx().GetChainStorage().UpdGroup(bft.producer.grpItem)
	}

	//check if need continue propose
	trxBufLen, err := bft.txBuffer.GetBufferLen()
	if err != nil {
		trx_bft_log.Warnf(err.Error())
	}

	trx_bft_log.Debugf("<%s> remove finished acs inst <%d>", bft.producer.groupId, epoch)
	bft.acsInsts.Delete(epoch)

	trx_bft_log.Debugf("<%s> After propose, trx buffer length <%d>", bft.producer.groupId, trxBufLen)
	//start next round
	if trxBufLen != 0 {
		newEpoch := bft.producer.cIface.GetCurrEpoch() + 1
		trx_bft_log.Debugf("<%s> try propose with new Epoch <%d>", bft.producer.groupId, newEpoch)
		bft.propose(newEpoch)
	}
}

func (bft *TrxBft) buildBlock(epoch int64, trxs map[string]*quorumpb.Trx) error {
	trx_bft_log.Infof("<%s> buildBlock for epoch <%d>", bft.producer.groupId, epoch)
	//try build block by using trxs
	trxToPackage := bft.sortTrx(trxs)

	parentEpoch := epoch - 1
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(bft.producer.groupId, parentEpoch, false, bft.producer.nodename)
	if err != nil {
		trx_acs_log.Warnf("?????????????????????????? GetBlock failed <%s>", err.Error())
		return err
	}

	//TBD fill withnesses
	witnesses := []*quorumpb.Witnesses{}

	//create block
	ks := localcrypto.GetKeystore()
	sudo := false
	newBlock, err := rumchaindata.CreateBlockByEthKey(parent, epoch, trxToPackage, sudo, bft.producer.grpItem.UserSignPubkey, witnesses, ks, "", bft.producer.nodename)
	if err != nil {
		trx_bft_log.Warnf("?????????????????????????? CreateBlockByEthKey failed <%s>", err.Error())
		trx_bft_log.Warnf("?????????????????????????? parent block <%v>", parent)
		return err
	}

	trx_bft_log.Info("molassproducer handle block just built")
	err = nodectx.GetNodeCtx().GetChainStorage().AddBlock(newBlock, false, bft.producer.nodename)
	if err != nil {
		return err
	}

	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		//approved producers
		bft.producer.cIface.ApplyTrxsProducerNode(trxToPackage, bft.producer.nodename)
	} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
		//owner
		bft.producer.cIface.ApplyTrxsFullNode(trxToPackage, bft.producer.nodename)
	}

	//broadcast new block
	trx_bft_log.Infof("<%s> broadcast new block to user channel", bft.producer.groupId)
	connMgr, err := conn.GetConn().GetConnMgr(bft.producer.groupId)
	if err != nil {
		return err
	}
	err = connMgr.BroadcastBlock(newBlock)
	if err != nil {
		trx_acs_log.Warnf("<%s> BroadcastBlock failed <%s>", bft.producer.groupId, err.Error())
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

func (bft *TrxBft) propose(epoch int64) error {
	trx_bft_log.Debugf("<%s> propose called, epoch <%d>", bft.producer.groupId, epoch)

	trxs, err := bft.txBuffer.GetNRandTrx(bft.BatchSize)
	if err != nil {
		return err
	}

	//nothing to propose
	if len(trxs) == 0 {
		trx_acs_log.Infof("trx queue empty, nothing to propose")
		return nil
	} else {
		for _, trx := range trxs {
			trx_acs_log.Debugf("try packageing trx <%s>", trx.TrxId)
		}
	}

	trxBundle := &quorumpb.HBTrxBundle{}
	trxBundle.Trxs = append(trxBundle.Trxs, trxs...)

	datab, err := proto.Marshal(trxBundle)
	if err != nil {
		return err
	}

	var acs *TrxACS
	inst, ok := bft.acsInsts.Load(epoch)
	if !ok {
		acs = NewTrxACS(bft.Config, bft, epoch)
		bft.acsInsts.Store(epoch, acs)
	} else {
		acs = inst.(*TrxACS)
	}

	return acs.InputValue(datab)
}

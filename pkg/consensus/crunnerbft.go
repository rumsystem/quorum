package consensus

import (
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var cr_bft_log = logging.Logger("cbft")

type CrBft struct {
	Config
	crunner  *CRunner
	acsInsts sync.Map //[int64]*TrxACS //map key is epoch
	txBuffer *TrxBuffer
	ptx      map[string]bool
}

func NewCrBft(cfg Config, crunner *CRunner) *CrBft {
	cr_bft_log.Debugf("<%s> NewCrBft called", crunner.groupId)
	return &CrBft{
		Config:   cfg,
		crunner:  crunner,
		txBuffer: NewTrxBuffer(crunner.groupId),
		ptx:      make(map[string]bool),
	}
}

func (bft *CrBft) AddTrx(tx *quorumpb.Trx) error {
	cr_bft_log.Debugf("<%s> AddTrx called", bft.crunner.groupId)
	bft.txBuffer.Push(tx)

	found := false
	f := func(key, value any) bool {
		TopEpoch := bft.crunner.cIface.GetCurrEpoch() + 1 //proposed but not finished epoch is current group epoch + 1 (next epoch)
		if key == TopEpoch {
			found = true
		}
		return true
	}

	bft.acsInsts.Range(f)
	if found {
		cr_bft_log.Debugf("<%s> Trx saved to TrxBuffer, wait to be propose", tx.TrxId)
		return nil
	}

	//try propose with next epoch
	newEpoch := bft.crunner.cIface.GetCurrEpoch() + 1
	cr_bft_log.Debugf("Try propose with new Epoch <%d>", newEpoch)
	bft.propose(newEpoch)
	return nil
}

func (bft *CrBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	cr_bft_log.Debugf("<%s> HandleMessage called, Epoch <%d>", bft.crunner.groupId, hbmsg.Epoch)
	var acs *CrACS
	inst, ok := bft.acsInsts.Load(hbmsg.Epoch)

	if !ok {
		if hbmsg.Epoch <= bft.crunner.cIface.GetCurrEpoch() {
			cr_bft_log.Warnf("message from old epoch, ignore")
			return nil
		}
		//create newTrxAcs and save it
		acs = NewCrACS(bft.Config, bft, hbmsg.Epoch)
		//TrxACS will be cast by syncmap to type ANY automatically
		bft.acsInsts.Store(hbmsg.Epoch, acs)
		cr_bft_log.Debugf("Create new ACS %d", hbmsg.Epoch)
	} else {
		//get acs from syncmap, cast from type ANY back to TrxAcs
		acs = inst.(*CrACS)
	}

	return acs.HandleMessage(hbmsg)
}

func (bft *CrBft) AcsDone(epoch int64, result map[string][]byte) {
	cr_bft_log.Debugf("<%s> AcsDone called, Epoch <%d>", bft.crunner.groupId, epoch)
	trxs := make(map[string]*quorumpb.Trx) //trx_id

	//decode trxs
	for key, value := range result {
		trxBundle := &quorumpb.HBTrxBundle{}
		err := proto.Unmarshal(value, trxBundle)
		if err != nil {
			cr_bft_log.Warningf("decode trxs failed for rbc inst %s, err %s", key, err.Error())
			return
		}

		for _, trx := range trxBundle.Trxs {
			if _, ok := trxs[trx.TrxId]; !ok {
				trxs[trx.TrxId] = trx
			}
		}
	}

	trxToPackage := bft.sortTrx(trxs)
	for _, trx := range trxToPackage {
		//for log analysiser
		cr_bft_log.Debugf("Epoch <%d> Packaging trx <%s>", epoch, trx.TrxId)
	}

	//update chain info
	bft.crunner.cIface.IncCurrEpoch()
	bft.crunner.cIface.SetLastUpdate(time.Now().UnixNano())

	//remove all trx
	for _, trx := range trxToPackage {
		err := bft.txBuffer.Delete(trx.TrxId)
		trx_bft_log.Debugf("remove packaged trx <%s>", epoch-1, trx.TrxId)
		if err != nil {
			trx_bft_log.Warnf(err.Error())
		}

		//save it to packaged trxid map
		if ok := bft.ptx[trx.TrxId]; ok {
			trx_bft_log.Warnf("trxid <%s> repackaged", trx.TrxId)
		}
		bft.ptx[trx.TrxId] = true
	}

	//check if need continue propose
	trxBufLen, err := bft.txBuffer.GetBufferLen()
	if err != nil {
		cr_bft_log.Warnf(err.Error())
	}

	cr_bft_log.Debugf("<%s> remove finished acs inst <%d>", bft.crunner.groupId, epoch)
	bft.acsInsts.Delete(epoch)

	cr_bft_log.Debugf("<%s> After propose, trx buffer length <%d>", bft.crunner.groupId, trxBufLen)
	//start next round
	if trxBufLen != 0 {
		newEpoch := bft.crunner.cIface.GetCurrEpoch() + 1
		cr_bft_log.Debugf("<%s> try propose with new Epoch <%d>", bft.crunner.groupId, newEpoch)
		bft.propose(newEpoch)
	}
}

func (bft *CrBft) sortTrx(trxs map[string]*quorumpb.Trx) []*quorumpb.Trx {
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
		if key == bft.crunner.grpItem.OwnerPubKey {
			continue
		}
		//append
		result = append(result, container[key]...)
	}

	//append any trxs from owner at the end of trxs slice
	if ownertrxs, ok := container[bft.crunner.grpItem.OwnerPubKey]; ok {
		result = append(result, ownertrxs...)
	}

	return result
}

func (bft *CrBft) propose(epoch int64) error {
	cr_bft_log.Debugf("<%s> propose called, epoch <%d>", bft.crunner.groupId, epoch)

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

	var acs *CrACS
	inst, ok := bft.acsInsts.Load(epoch)
	if !ok {
		acs = NewCrACS(bft.Config, bft, epoch)
		bft.acsInsts.Store(epoch, acs)
	} else {
		acs = inst.(*CrACS)
	}

	return acs.InputValue(datab)
}

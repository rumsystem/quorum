package consensus

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var bft_log = logging.Logger("tbft")

type TrxBft struct {
	Config
	producer     *MolassesProducer
	acsInsts     map[int64]*ACS //map key is epoch
	txBuffer     *TrxBuffer
	sudoTxBuffer *TrxBuffer
}

func NewTrxBft(cfg Config, producer *MolassesProducer) *TrxBft {
	bft_log.Debugf("NewBft called")
	return &TrxBft{
		Config:   cfg,
		producer: producer,
		acsInsts: make(map[int64]*ACS),
		txBuffer: NewTrxBuffer(producer.groupId),
	}
}

func (bft *TrxBft) AddTrx(tx *quorumpb.Trx) error {
	bft_log.Debugf("AddTrx called")
	//bft_log.Debugf("IsSudoTrx : <%v>", tx.SudoTrx)
	bft.txBuffer.Push(tx)
	newEpoch := bft.producer.grpItem.Epoch + 1
	bft_log.Debugf("Try propose with new Epoch <%d>", newEpoch)
	bft.propose(newEpoch)
	return nil
}

func (bft *TrxBft) AddSudoTrx(tx *quorumpb.Trx) error {
	bft_log.Debugf("AddSudoTrx called")

	//check if sudotrx is from group owner
	if bft.producer.grpItem.OwnerPubKey != tx.SenderPubkey {
		bft_log.Warnf("SudoTrx <%s> from non owner <%s>, ignore", tx.TrxId, tx.SenderPubkey)
		return nil
	}

	//check if I am owner
	if bft.producer.grpItem.OwnerPubKey != bft.producer.grpItem.UserSignPubkey {
		bft_log.Warnf("Ignore by me, owner node will handle sudotrx <%s>", tx.SenderPubkey)
		return nil
	}

	//SudoTrx will bypass consensus, owner node will generate SudoBlock by itself

	return nil
}

func (bft *TrxBft) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	bft_log.Debugf("HandleMessage called, Epoch <%d>", hbmsg.Epoch)
	acs, ok := bft.acsInsts[hbmsg.Epoch]

	if !ok {
		if hbmsg.Epoch <= bft.producer.grpItem.Epoch {
			bft_log.Warnf("message from old epoch, ignore")
			return nil
		}
		acs = NewACS(bft.Config, bft, hbmsg.Epoch)
		bft.acsInsts[hbmsg.Epoch] = acs
		bft_log.Debugf("Create new ACS %d", hbmsg.Epoch)
	}

	return acs.HandleMessage(hbmsg)
}

func (hb *TrxBft) AcsDone(epoch int64, result map[string][]byte) {
	bft_log.Debugf("AcsDone called, Epoch <%d>", epoch)
	trxs := make(map[string]*quorumpb.Trx) //trx_id
	//bft_log.Infof("result %v", result)

	//decode trxs
	for key, value := range result {
		trxBundle := &quorumpb.HBTrxBundle{}
		bft_log.Infof("raw TrxBundle %v", value)
		err := proto.Unmarshal(value, trxBundle)
		if err != nil {
			bft_log.Warningf("decode trxs failed for rbc inst %s, err %s", key, err.Error())
			value = value[:len(value)-1]
			err := proto.Unmarshal(value, trxBundle)
			if err != nil {
				bft_log.Warningf("decode trxs still failed for rbc inst %s, err %s", key, err.Error())
			}
		}

		for _, trx := range trxBundle.Trxs {
			if _, ok := trxs[trx.TrxId]; !ok {
				trxs[trx.TrxId] = trx
			}
		}
	}

	//TBD order trx
	err := hb.buildBlock(epoch, trxs)
	if err != nil {
		bft_log.Warnf(err.Error())
	}

	//remove outputed trxs from buffer
	for trxId, _ := range trxs {
		err := hb.txBuffer.Delete(trxId)
		if err != nil {
			bft_log.Warnf(err.Error())
		}
	}

	//clear acs for finished epoch
	hb.acsInsts[epoch] = nil
	delete(hb.acsInsts, epoch)

	bft_log.Debugf("Remove acs %d", epoch)

	bft_log.Debugf("<%s> UpdChainInfo called", hb.producer.groupId)
	hb.producer.grpItem.Epoch = epoch
	hb.producer.grpItem.LastUpdate = time.Now().Unix()
	bft_log.Infof("<%s> Chain Info updated, epoch %d", hb.producer.groupId, epoch)
	nodectx.GetNodeCtx().GetChainStorage().UpdGroup(hb.producer.grpItem)

	trxBufLen, err := hb.txBuffer.GetBufferLen()
	if err != nil {
		bft_log.Warnf(err.Error())
	}

	bft_log.Debugf("After propose, trx buffer length %d", trxBufLen)

	//start next round
	if trxBufLen != 0 {
		newEpoch := hb.producer.grpItem.Epoch + 1
		bft_log.Debugf("Try propose with new Epoch <%d>", newEpoch)
		hb.propose(newEpoch)
	}
}

func (hb *TrxBft) buildBlock(epoch int64, trxs map[string]*quorumpb.Trx) error {
	//try build block by using trxs

	var trxToPackage []*quorumpb.Trx
	bft_log.Infof("---------------acs result for epoch %d-------------------", epoch)

	for trxId, trx := range trxs {
		bft_log.Infof(">>> trxId : %s", trxId)
		trxToPackage = append(trxToPackage, trx)
	}

	//update db here
	parentEpoch := epoch - 1
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(hb.producer.groupId, parentEpoch, false, hb.producer.nodename)
	if err != nil {
		return err
	}

	//TBD fill withnesses
	witnesses := []*quorumpb.Witnesses{}

	//create block
	ks := localcrypto.GetKeystore()
	sudo := false
	newBlock, err := rumchaindata.CreateBlockByEthKey(parent, epoch, trxToPackage, sudo, hb.producer.grpItem.UserSignPubkey, witnesses, ks, "", hb.producer.nodename)
	if err != nil {
		return err
	}

	//broadcast new block
	connMgr, err := conn.GetConn().GetConnMgr(hb.producer.groupId)
	if err != nil {
		return err
	}
	err = connMgr.SendBlockPsconn(newBlock, conn.UserChannel)
	if err != nil {
		acs_log.Warnf("<%s> <%s>", hb.producer.groupId, err.Error())
	}

	//if run as producer node
	if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
		bft_log.Info("PRODUCER_NODE handle block")
		hb.producer.cIface.ApplyTrxsProducerNode(trxToPackage, hb.producer.nodename)
		err := nodectx.GetNodeCtx().GetChainStorage().AddBlock(newBlock, false, hb.producer.nodename)
		if err != nil {
			return err
		}
	} else {
		// if run in FULL_NODE, no need to handle this block here
		// local user will receive this block via producer channel, local user will handle it
		bft_log.Info("FULL_NODE handle block, do nothing, wait for local user to handle it")
	}

	return nil
}

func (hb *TrxBft) propose(epoch int64) error {
	trxs, err := hb.txBuffer.GetNRandTrx(hb.BatchSize)
	if err != nil {
		return err
	}

	//nothing to propose
	if len(trxs) == 0 {
		acs_log.Infof("trx queue empty, nothing to propose")
		return nil
	}

	trxBundle := &quorumpb.HBTrxBundle{}
	for _, trx := range trxs {
		trxBundle.Trxs = append(trxBundle.Trxs, trx)
	}

	datab, err := proto.Marshal(trxBundle)
	acs, ok := hb.acsInsts[epoch]
	if !ok {
		acs = NewACS(hb.Config, hb, epoch)
		hb.acsInsts[epoch] = acs
	}

	return hb.acsInsts[epoch].InputValue(datab)
}

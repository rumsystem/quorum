package consensus

import (
	"github.com/golang/protobuf/proto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var bft_log = logging.Logger("bft")

type Bft struct {
	Config
	producer *MolassesProducer
	epoch    int64          //current epoch
	acsInsts map[int64]*ACS //map key is epoch
	txBuffer *TrxBuffer
	outputs  map[int64][]*quorumpb.Trx
}

func NewBft(cfg Config, producer *MolassesProducer) *Bft {
	bft_log.Debugf("NewBft called")
	return &Bft{
		Config:   cfg,
		producer: producer,
		epoch:    producer.grpItem.Epoch,
		acsInsts: make(map[int64]*ACS),
		txBuffer: NewTrxBuffer(producer.groupId),
		outputs:  make(map[int64][]*quorumpb.Trx),
	}
}

func (bft *Bft) AddTrx(tx *quorumpb.Trx) error {
	bft_log.Debugf("AddTrx called")
	bft.txBuffer.Push(tx)
	len, err := bft.txBuffer.GetBufferLen()
	if err != nil {
		return err
	}

	bft_log.Infof("trx buffer len %d", len)
	bft.propose()

	return nil
}

func (bft *Bft) HandleMessage(hbmsg *quorumpb.HBMsg) error {
	bft_log.Debugf("HandleMessage called")
	acs, ok := bft.acsInsts[hbmsg.Epoch]

	if !ok {
		if hbmsg.Epoch < bft.epoch {
			bft_log.Warnf("message from old epoch, ignore")
			return nil
		}

		acs = NewACS(bft.Config, bft, hbmsg.Epoch)
		bft.acsInsts[hbmsg.Epoch] = acs
	}

	return acs.HandleMessage(hbmsg)
}

func (hb *Bft) AcsDone(epoch int64, result map[string][]byte) {
	bft_log.Debugf("AcsDone called %d", epoch)
	var trxs map[string]*quorumpb.Trx
	trxs = make(map[string]*quorumpb.Trx) //trx_id

	//decode trxs
	for key, value := range result {
		trxBundle := &quorumpb.HBTrxBundle{}
		err := proto.Unmarshal(value, trxBundle)
		if err != nil {
			bft_log.Warningf("decode trxs failed for rbc inst %s", key)
		} else {
			for _, trx := range trxBundle.Trxs {
				if _, ok := trxs[trx.TrxId]; !ok {
					trxs[trx.TrxId] = trx
				}
			}
		}
	}
	//order trx

	err := hb.buildBlock(trxs)
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

	//advanced to next epoch
	hb.epoch++
	bft_log.Debugf("advance to epoch  %d", hb.epoch)

	//Don't update chain Info here
	/*
		bft_log.Debugf("<%s> UpdChainInfo called", hb.producer.groupId)
		hb.producer.grpItem.Epoch = hb.epoch
		hb.producer.grpItem.LastUpdate = time.Now().Unix()
		bft_log.Infof("<%s> Chain Info updated, epoch %d", hb.producer.groupId, hb.epoch)
		nodectx.GetNodeCtx().GetChainStorage().UpdGroup(hb.producer.grpItem)
	*/
	trxBufLen, err := hb.txBuffer.GetBufferLen()
	if err != nil {
		bft_log.Warnf(err.Error())
	}

	bft_log.Debugf("After propose, trx buffer length %d", trxBufLen)

	//start next round
	if trxBufLen != 0 {
		hb.propose()
	}
}

func (hb *Bft) buildBlock(trxs map[string]*quorumpb.Trx) error {
	//try build block by using trxs

	var trxToPackage []*quorumpb.Trx
	bft_log.Infof("---------------acs result for epoch %d-------------------", hb.epoch)

	for trxId, trx := range trxs {
		bft_log.Infof(">>> trxId : %s", trxId)
		trxToPackage = append(trxToPackage, trx)
	}

	//update db here
	parent, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(hb.producer.groupId, hb.epoch-1, false, hb.producer.nodename)
	if err != nil {
		return err
	}

	//TBD fill withnesses
	witnesses := []*quorumpb.Witnesses{}

	//create block
	ks := localcrypto.GetKeystore()
	newBlock, err := rumchaindata.CreateBlockByEthKey(parent, hb.epoch, trxToPackage, hb.producer.grpItem.UserSignPubkey, witnesses, ks, "", hb.producer.nodename)
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

func (hb *Bft) propose() error {
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
	acs, ok := hb.acsInsts[hb.epoch]
	if !ok {
		acs = NewACS(hb.Config, hb, hb.epoch)
		hb.acsInsts[hb.epoch] = acs
	}

	return hb.acsInsts[hb.epoch].InputValue(datab)
}

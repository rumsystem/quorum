package consensus

import (
	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var bft_log = logging.Logger("bft")

type Bft struct {
	Config
	groupId  string
	acsInsts map[uint64]*ACS //map key is epoch
	txBuffer *TrxBuffer
	epoch    uint64 //current epoch
	outputs  map[uint64][]*quorumpb.Trx
}

func NewBft(cfg Config, groupId string) *Bft {
	return &Bft{
		Config:   cfg,
		groupId:  groupId,
		acsInsts: make(map[uint64]*ACS),
		txBuffer: NewTrxBuffer(groupId),
		outputs:  make(map[uint64][]*quorumpb.Trx),
	}
}

func (bft *Bft) AddTrx(tx *quorumpb.Trx) error {
	bft.txBuffer.Push(tx)
	len, err := bft.txBuffer.GetBufferLen()
	if err != nil {
		return err
	}

	//start produce
	if len == 1 {
		bft.propose()
	}

	return nil
}

func (bft *Bft) HandleMessage(senderId string, epoch uint64, msg *quorumpb.HBMsg) error {
	acs, ok := bft.acsInsts[epoch]
	if !ok {
		if epoch < bft.epoch {
			bft_log.Warnf("message from old epoch, ignore")
			return nil
		}

		acs = NewACS(bft.Config, bft, epoch)
		bft.acsInsts[epoch] = acs
	}

	if err := acs.HandleMessage(msg); err != nil {
		return err
	}

	return nil
}

func (hb *Bft) AcsDone(epoch uint64, result map[string][]byte) {
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
		acs_log.Warnf(err.Error())
	}

	//remove outputed trxs from buffer
	for trxId, _ := range trxs {
		err := hb.txBuffer.Delete(trxId)
		if err != nil {
			acs_log.Warnf(err.Error())
		}
	}

	//clear acs for finished epoch
	hb.acsInsts[epoch] = nil

	//advanced to next epoch
	hb.epoch++

	trxBufLen, err := hb.txBuffer.GetBufferLen()
	if err != nil {
		acs_log.Warnf(err.Error())
	}

	//start next round
	if trxBufLen != 0 {
		hb.propose()
	}
}

func (hb *Bft) buildBlock(trxs map[string]*quorumpb.Trx) error {
	//try build block by using trxs
	acs_log.Infof("------------------------------------------")
	acs_log.Infof("acs result for epoch %d", hb.epoch)

	for trxId, _ := range trxs {
		acs_log.Infof("%s", trxId)
	}

	return nil
}

func (hb *Bft) propose() error {
	trxs, err := hb.txBuffer.GetNRandTrx(hb.BatchSize)
	if err != nil {
		return err
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

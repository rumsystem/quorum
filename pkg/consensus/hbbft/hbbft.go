package hbbft

import (
	"math"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var hbbft_log = logging.Logger("hbbft")

type HBMessage struct {
	Epoch   uint64
	Payload interface{}
}

type HoneyBadger struct {
	groupId string
	Config
	acsInsts map[uint64]*ACS //map key is epoch
	txBuffer *TrxBuffer
	epoch    uint64	//current epoch

	lock    sync.RWMutex
	outputs map[uint64][]HBTrx

	messageQue *messageQue
	msgCount   int
}

func NewHB(cfg Config, groupId string) *HoneyBadger {
	return &HoneyBadger{
		Config:     cfg,
		groupId:  
		acsInsts:   make(map[uint64]*ACS),
		txBuffer:   NewTrxBuffer(),
		outputs:    make(map[uint64][]HBTrx),
		messageQue: newMessageQue(),
	}
}

func (hb *HoneyBadger) Messages() []MessageTuple {
	return hb.messageQue.messages()
}

func (hb *HoneyBadger) AddTrx(tx *quorumpb.Trx) error {
	hb.txBuffer.Push(tx)
	return nil
}

func (hb *HoneyBadger) HandleMessage(senderId string, epoch uint64, msg *ACSMessage) error {
	hb.msgCount++
	acs, ok := hb.acsInsts[epoch]
	if !ok {
		if epoch < hb.epoch {
			hbbft_log.Warnf("message from old epoch, ignore")
			return nil
		}

		acs = NewACS(hb.Config)
		hb.acsInsts[epoch] = acs
	}

	if err := acs.handleMessage(senderId, msg); err != nil {
		return err
	}

	hb.addMessage(acs.messageQue.messages())
	if hb.epoch == epoch {
		return hb.maybeProcessOutput()
	}

	hb.removeOldEpochs(epoch)
}

func (hb *HoneyBadger) Start() error {
	return hb.propose()
}

func (hb *HoneyBadger) lenMempool() int {
	return int(hb.txBuffer.length)
}

func (hb *HoneyBadger) propose() error {
	if hb.txBuffer.length == 0 {
		time.Sleep(2 * time.Second)
		return hb.propose()
	}

	batchSize := hb.BatchSize
	if batchSize == 0 {
		scalar := 20
		batchSize
	}

	batchSize = int(math.Min(float64(batchSize), float64(hb.txBuffer.length)))
	n := int(math.Max(float64(1), float64(batchSize/len(hb.Nodes))))
	batch := sample(hb.txBuffer.data[:batchSize], n)

}

func (hb *HoneyBadger) getOrNewAcsInst(epoch uint64) *ACS {
	if acs, ok := hb.acsInsts[epoch]; ok {
		return acs
	}

	acs := NewACS(hb.Config)
	hb.acsInsts[epoch] = acs
	return acs
}

func (hb *HoneyBadger) addMessage(msgs []MessageTuple) {
	for _, msg := range msgs {
		hb.messageQue.addMessage(HBMessage{hb.epoch, msg.Payload}, msg.To)
	}
}

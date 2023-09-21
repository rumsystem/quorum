package consensus

import (
	"context"
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var molaproducer_log = logging.Logger("producer")

type MolassesProducer struct {
	groupId  string
	nodename string
	grpItem  *quorumpb.GroupItem
	cIface   def.ChainMolassesIface

	ptbft  *PTBft
	ctx    context.Context
	locker sync.RWMutex
}

func (producer *MolassesProducer) NewProducer(ctx context.Context, item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molaproducer_log.Debugf("<%s> NewProducer called", item.GroupId)
	producer.nodename = nodename
	producer.groupId = item.GroupId
	producer.grpItem = item
	producer.cIface = iface
	producer.ctx = ctx
}

func (producer *MolassesProducer) StartPropose() {
	molaproducer_log.Debugf("<%s> StartPropose called", producer.groupId)

	producer.locker.Lock()
	defer producer.locker.Unlock()

	if !producer.cIface.IsProducer() {
		molaproducer_log.Debugf("<%s> unapproved producer do nothing", producer.groupId)
		return
	}

	producerPubkey := producer.cIface.GetMyProducerPubkey()

	molaproducer_log.Debugf("<%s> producer <%s> start propose", producer.groupId, producerPubkey)
	config, err := producer.createBftConfig(producerPubkey)
	if err != nil {
		molaproducer_log.Error("create bft failed with error: %s", err.Error())
		return
	}

	producer.ptbft = NewPTBft(producer.ctx, *config, producer.cIface)
	producer.ptbft.Start()
}

func (producer *MolassesProducer) StopPropose() {
	molaproducer_log.Debug("StopPropose called")
	producer.locker.Lock()
	defer producer.locker.Unlock()

	if producer.ptbft != nil {
		producer.ptbft.Stop()
	}
	producer.ptbft = nil
}

func (producer *MolassesProducer) createBftConfig(producerPubkey string) (*Config, error) {
	molaproducer_log.Debugf("<%s> createBftConfig called", producer.groupId)
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(producer.groupId, producer.nodename)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, producer := range producer_nodes {
		nodes = append(nodes, producer.ProducerPubkey)
	}

	molaproducer_log.Debugf("Get <%d> producers", len(nodes))
	for _, producerId := range nodes {
		molaproducer_log.Debugf(">>> producer_id <%s>", producerId)
	}

	N := len(nodes)
	f := (N - 1) / 3 //f * 3 < N

	molaproducer_log.Debugf("Failable node <%d>", f)

	//use fixed scalar size
	scalar := 20
	//batchSize := (len(nodes) * 2) * scalar
	batchSize := scalar

	molaproducer_log.Debugf("batchSize <%d>", batchSize)

	//get producer keyname
	keyname := producer.cIface.GetKeynameByPubkey(producerPubkey)
	if keyname == "" {
		molaproducer_log.Debugf("get keyname failed")
		return nil, nil
	}

	config := &Config{
		GroupId:     producer.groupId,
		NodeName:    producer.nodename,
		MyPubkey:    producerPubkey,
		MyKeyName:   keyname,
		OwnerPubKey: producer.grpItem.OwnerPubKey,
		N:           N,
		f:           f,
		Nodes:       nodes,
		BatchSize:   batchSize,
	}

	return config, nil
}

func (producer *MolassesProducer) AddTrxToTxBuffer(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err := producer.ptbft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed with error <%s>", err.Error())
	}
}

func (producer *MolassesProducer) HandleBftMsg(bftMsg *quorumpb.BftMsg) error {
	//molaproducer_log.Debugf("<%s> HandleBFTMsg called", producer.groupId)
	if bftMsg.Type == quorumpb.BftMsgType_HB_BFT {
		//unmarshal bft msg
		hbMsg := &quorumpb.HBMsgv1{}
		err := proto.Unmarshal(bftMsg.Data, hbMsg)
		if err != nil {
			molaproducer_log.Errorf("unmarshal bft msg failed with error: %s", err.Error())
			return err
		}

		if producer.ptbft != nil {
			producer.ptbft.HandleHBMessage(hbMsg)
		}
	}

	return nil
}

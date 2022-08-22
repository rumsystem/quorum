package consensus

import (

	//p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var molaproducer_log = logging.Logger("producer")

type MolassesProducer struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
	bft      *Bft
}

func (producer *MolassesProducer) Init(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molaproducer_log.Debug("Init called")
	producer.grpItem = item
	producer.cIface = iface
	producer.nodename = nodename
	producer.groupId = item.GroupId

	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Errorf("create bft failed")
		molauser_log.Error(err.Error())
		return
	}

	producer.bft = NewBft(*config, producer)
	producer.bft.propose()

	molaproducer_log.Infof("<%s> producer created", producer.groupId)
}

func (producer *MolassesProducer) createBftConfig() (*Config, error) {
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(producer.groupId, producer.nodename)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, producer := range producer_nodes {
		nodes = append(nodes, producer.ProducerPubkey)
	}

	n := len(nodes)
	f := (n - 1) / 2

	scalar := 20
	batchSize := (len(nodes) * 2) * scalar

	config := &Config{
		N:            n,
		F:            f,
		Nodes:        nodes,
		BatchSize:    batchSize,
		MySignPubkey: producer.grpItem.UserSignPubkey,
	}

	return config, nil
}

// Add trx to trx pool
func (producer *MolassesProducer) AddTrx(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> AddTrx called", producer.groupId)

	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, producer.nodename)
	if err != nil {
		return
	}

	if !isAllow {
		molaproducer_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", producer.groupId, trx.SenderPubkey, trx.Type.String())
		return
	}

	//check if trx with same nonce exist, !!Only applied to client which support nonce
	isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.TrxId, trx.Nonce, producer.nodename)
	if isExist {
		molaproducer_log.Debugf("<%s> Trx <%s> with nonce <%d> already packaged, ignore <%s>", producer.groupId, trx.TrxId, trx.Nonce)
		return
	}

	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err = producer.bft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed %s", err.Error())
	}
}

func (producer *MolassesProducer) HandleHBMsg(hbmsg *quorumpb.HBMsg) error {
	molaproducer_log.Debugf("<%s> HandleHBMsg %s", producer.groupId, hbmsg.MsgType.String())
	return producer.bft.HandleMessage(hbmsg)
}

package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var crunner_log = logging.Logger("crunner")

type CRunner struct {
	grpItem  *quorumpb.GroupItem
	groupId  string
	nodename string
	bft      *CrBft
	cIface   def.ChainMolassesIface
}

func (crunner *CRunner) NewCRunner(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	crunner_log.Debug("NewCRunner called")
	crunner.grpItem = item
	crunner.cIface = iface
	crunner.nodename = nodename
	crunner.groupId = item.GroupId

	config, err := crunner.createBftConfig()
	if err != nil {
		crunner_log.Error("create bft failed")
		crunner_log.Error(err.Error())
		return
	}
	crunner.bft = NewCrBft(*config, crunner)
}

func (crunner *CRunner) IsTrxPackaged(trxId string) bool {
	return crunner.bft.ptx[trxId]
}

func (crunner *CRunner) createBftConfig() (*Config, error) {
	crunner_log.Debugf("<%s> createBftConfig called", crunner.groupId)
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(crunner.groupId, crunner.nodename)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, producer := range producer_nodes {
		nodes = append(nodes, producer.ProducerPubkey)
	}

	crunner_log.Debugf("Get <%d> producers", len(nodes))
	for _, producerId := range nodes {
		crunner_log.Debugf(">>> producer_id <%s>", producerId)
	}

	N := len(nodes)
	f := (N - 1) / 3 //f * 3 < N

	crunner_log.Debugf("Failable node <%d>", f)

	//use fixed scalar size
	scalar := 20
	//batchSize := (len(nodes) * 2) * scalar
	batchSize := scalar

	crunner_log.Debugf("batchSize <%d>", batchSize)

	config := &Config{
		N:         N,
		f:         f,
		Nodes:     nodes,
		BatchSize: batchSize,
		MyPubkey:  crunner.grpItem.UserSignPubkey,
	}

	return config, nil
}

func (crunner *CRunner) TryPropose() {
	crunner_log.Debug("TryPropose called")
	newEpoch := crunner.cIface.GetCurrEpoch() + 1
	crunner.bft.propose(newEpoch)
}

func (crunner *CRunner) AddTrx(trx *quorumpb.Trx) {
	crunner_log.Debugf("<%s> crunner AddTrx called, add trx <%s>", crunner.groupId, trx.TrxId)
	err := crunner.bft.AddTrx(trx)
	if err != nil {
		crunner_log.Errorf("add trx failed %s", err.Error())
	}
}

func (crunner *CRunner) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//crunner_log.Debugf("<%s> HandleHBMsg, Epoch <%d>", producer.groupId, hbmsg.Epoch)
	return crunner.bft.HandleMessage(hbmsg)
}

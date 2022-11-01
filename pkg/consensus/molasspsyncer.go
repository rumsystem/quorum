package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var molapsyncer_log = logging.Logger("psyncer")

type MolassesPSyncer struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
	bft      *PSyncBft
}

func (psyncer *MolassesPSyncer) NewPSyncer(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molapsyncer_log.Debugf("<%s> NewPSyncer called", item.GroupId)
	psyncer.grpItem = item
	psyncer.cIface = iface
	psyncer.nodename = nodename
	psyncer.groupId = item.GroupId

	config, err := psyncer.createBftConfig()
	if err != nil {
		molapsyncer_log.Error("create bft failed")
		molapsyncer_log.Error(err.Error)
		return
	}

	psyncer.bft = NewPSyncBft(*config, psyncer)
}

func (psyncer *MolassesPSyncer) RecreateBft() {
	molapsyncer_log.Debugf("<%s> RecreateBft called", psyncer.groupId)
	config, err := psyncer.createBftConfig()
	if err != nil {
		molapsyncer_log.Errorf("recreate bft failed")
		molapsyncer_log.Error(err.Error())
		return
	}

	psyncer.bft = NewPSyncBft(*config, psyncer)
}

func (psyncer *MolassesPSyncer) TryPropose() {
	molapsyncer_log.Debugf("<%s> TryPropose called", psyncer.groupId)
	psyncer.bft.Propose()
}

func (psyncer *MolassesPSyncer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molapsyncer_log.Debugf("<%s> HandleHBMsg %s, %d", psyncer.groupId, hbmsg.MsgType.String(), hbmsg.Epoch)
	return psyncer.bft.HandleMessage(hbmsg)
}

func (psyncer *MolassesPSyncer) createBftConfig() (*Config, error) {
	molapsyncer_log.Debugf("<%s> createBftConfig called", psyncer.groupId)
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(psyncer.groupId, psyncer.nodename)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, producer := range producer_nodes {
		nodes = append(nodes, producer.ProducerPubkey)
	}

	molaproducer_log.Debugf("Get <%d> producers", len(nodes))
	for _, producerId := range nodes {
		molaproducer_log.Debugf(">>> producer_id %s", producerId)
	}

	n := len(nodes)
	f := (n - 1) / 2

	molaproducer_log.Debugf("Failable node %d", f)

	scalar := 20
	batchSize := (len(nodes) * 2) * scalar

	molaproducer_log.Debugf("batchSize %d", batchSize)

	config := &Config{
		N:            n,
		F:            f,
		Nodes:        nodes,
		BatchSize:    batchSize,
		MySignPubkey: psyncer.grpItem.UserSignPubkey,
	}

	return config, nil
}

package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var molapsyncer_log = logging.Logger("psyncer")

type MolassesPSync struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
	bft      *PSyncBft
}

func (psync *MolassesPSync) NewPSync(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molapsyncer_log.Debugf("<%s> NewPSyncer called", item.GroupId)
	psync.grpItem = item
	psync.cIface = iface
	psync.nodename = nodename
	psync.groupId = item.GroupId

	config, err := psync.createBftConfig()
	if err != nil {
		molapsyncer_log.Error("create bft failed")
		molapsyncer_log.Error(err.Error)
		return
	}

	psync.bft = NewPSyncBft(*config, psync)
}

func (psync *MolassesPSync) RecreateBft() {
	molapsyncer_log.Debugf("<%s> RecreateBft called", psync.groupId)
	config, err := psync.createBftConfig()
	if err != nil {
		molapsyncer_log.Errorf("recreate bft failed")
		molapsyncer_log.Error(err.Error())
		return
	}

	psync.bft = NewPSyncBft(*config, psync)
}

func (psync *MolassesPSync) AddConsensusReq(req *quorumpb.ConsensusMsg) error {
	molapsyncer_log.Debugf("<%s> TryGetConsensus called", psync.groupId)
	return psync.bft.AddConsensusReq(req)
}

func (psync *MolassesPSync) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molapsyncer_log.Debugf("<%s> HandleHBMsg %s, %d", psync.groupId, hbmsg.MsgType.String(), hbmsg.Epoch)
	return psync.bft.HandleMessage(hbmsg)
}

func (psync *MolassesPSync) createBftConfig() (*Config, error) {
	molapsyncer_log.Debugf("<%s> createBftConfig called", psync.groupId)
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(psync.groupId, psync.nodename)
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
		MySignPubkey: psync.grpItem.UserSignPubkey,
	}

	return config, nil
}

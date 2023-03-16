package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var molapp_log = logging.Logger("pp")

type MolassesProducerProposer struct {
	grpItem      *quorumpb.GroupItem
	groupId      string
	nodename     string
	cIface       def.ChainMolassesIface
	producers    []*quorumpb.ProducerItem
	trx          *quorumpb.Trx
	bft          *PPBft
	currReqId    string
	currReqNance int64
}

func (pp *MolassesProducerProposer) NewProducerProposer(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molapp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
}

func (pp *MolassesProducerProposer) RecreateBft() {
	molapp_log.Debugf("<%s> RecreateBft called", pp.groupId)
}

func (pp *MolassesProducerProposer) HandlePPREQ(req *quorumpb.ProducerProposalReq) {
	molapp_log.Debugf("<%s> HandlePPREQ called", pp.groupId)
}

func (pp *MolassesProducerProposer) HandleHBPP(hbmsg *quorumpb.HBMsgv1) {
	molapp_log.Debugf("<%s> HandleHBPP called", pp.groupId)
}

func (pp *MolassesProducerProposer) AddProposerItem(producerList *quorumpb.BFTProducerBundleItem, originalTrx *quorumpb.Trx, agrmTickCount, agrmTickLength, fromNewEpoch uint64) error {
	molapp_log.Debugf("<%s> AddProposerItem called", pp.groupId)
	return nil
}

func (pp *MolassesProducerProposer) createBftConfig() (*Config, error) {
	molapp_log.Debugf("<%s> createBftConfig called", pp.groupId)

	var producerNodes []string
	for _, producer := range pp.producers {
		molaproducer_log.Debugf(">>> producer <%s>", producer.ProducerPubkey)
		producerNodes = append(producerNodes, producer.ProducerPubkey)
	}

	n := len(producerNodes)
	f := (n - 1) / 3

	molaproducer_log.Debugf("failable producers <%d>", f)
	batchSize := 1

	config := &Config{
		N:         n,
		f:         f,
		Nodes:     producerNodes,
		BatchSize: batchSize,
		MyPubkey:  pp.grpItem.UserSignPubkey,
	}

	return config, nil
}

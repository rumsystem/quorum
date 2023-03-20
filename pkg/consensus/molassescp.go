package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var molacp_log = logging.Logger("cp")

type MolassesConsensusProposer struct {
	grpItem      *quorumpb.GroupItem
	groupId      string
	nodename     string
	cIface       def.ChainMolassesIface
	producers    []*quorumpb.ProducerItem
	trxId        string
	bft          *PCBft
	currReqId    string
	currReqNance int64

	fromNewEpoch    uint64
	trxEpochTickLen uint64

	ReqSender *CCReqSender
}

func (cp *MolassesConsensusProposer) NewConsensusProposer(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molacp_log.Debugf("<%s> NewProducerProposer called", item.GroupId)
}

func (cp *MolassesConsensusProposer) RecreateBft(agrmTickCount, agrmTickLength, fromNewEpoch, trxEpochTick uint64) {
	molacp_log.Debugf("<%s> RecreateBft called", cp.groupId)
}

func (cp *MolassesConsensusProposer) HandleCCReq(req *quorumpb.ChangeConsensusReq) error {
	molacp_log.Debugf("<%s> HandleCCReq called", cp.groupId)
	return nil
}

func (cp *MolassesConsensusProposer) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	molacp_log.Debugf("<%s> HandleHBPMsg called", cp.groupId)
	return nil
}

func (cp *MolassesConsensusProposer) AddCCItem(producerList *quorumpb.BFTProducerBundleItem, trxId string, agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen uint64) error {
	molacp_log.Debugf("<%s> AddCCItem called", cp.groupId)

	/*
		if cp.bft != nil {
			//pp.bft.Stop()
		}
	*/

	//create bft
	cp.RecreateBft(agrmTickLen, agrmTickCnt, fromNewEpoch, trxEpochTickLen)

	//add pubkeys for all producers
	cp.producers = append(cp.producers, producerList.Producers...)

	//save original trx (to propose new producers)
	cp.trxId = trxId

	//save agrm tick length
	cp.fromNewEpoch = fromNewEpoch

	//save trx epoch tick length
	cp.trxEpochTickLen = trxEpochTickLen

	//create change consensus req and start broadcast it

	return nil
}

func (pp *MolassesConsensusProposer) createBftConfig() (*Config, error) {
	molacp_log.Debugf("<%s> createBftConfig called", pp.groupId)

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

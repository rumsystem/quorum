package consensus

import (
	"github.com/rumsystem/quorum/pkg/consensus/def"
)

type Molasses struct {
	name              string
	producer          def.Producer
	user              def.User
	consensusProposer def.ConsensusProposer
}

func NewMolasses(p def.Producer, u def.User, cp def.ConsensusProposer) *Molasses {
	return &Molasses{name: "Molasses", producer: p, user: u, consensusProposer: cp}
}

func (m *Molasses) Name() string {
	return m.name
}

func (m *Molasses) Producer() def.Producer {
	return m.producer
}

func (m *Molasses) User() def.User {
	return m.user
}

func (m *Molasses) ConsensusProposer() def.ConsensusProposer {
	return m.consensusProposer
}

func (m *Molasses) SetProducer(p def.Producer) {
	m.producer = p
}

func (m *Molasses) SetUser(u def.User) {
	m.user = u
}

func (m *Molasses) SetConsensusProposer(pp def.ConsensusProposer) {
	m.consensusProposer = pp
}

func (m *Molasses) StartProposeTrx() {
	if m.producer != nil {
		m.producer.StartPropose()
	}
}

func (m *Molasses) StopProposeTrx() {
	if m.producer != nil {
		m.producer.StopPropose()
	}
}

package consensus

import (
	"github.com/rumsystem/quorum/pkg/consensus/def"
)

type Molasses struct {
	name             string
	producer         def.Producer
	user             def.User
	producerproposer def.ProducerProposer
}

func NewMolasses(p def.Producer, u def.User, pp def.ProducerProposer) *Molasses {
	return &Molasses{name: "Molasses", producer: p, user: u, producerproposer: pp}
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

func (m *Molasses) ProducerProposer() def.ProducerProposer {
	return m.producerproposer
}

func (m *Molasses) SetProducer(p def.Producer) {
	m.producer = p
}

func (m *Molasses) SetUser(u def.User) {
	m.user = u
}

func (m *Molasses) SetProducerProposer(pp def.ProducerProposer) {
	m.producerproposer = pp
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

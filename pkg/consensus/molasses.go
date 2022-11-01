package consensus

import (
	"github.com/rumsystem/quorum/pkg/consensus/def"
)

type Molasses struct {
	name     string
	producer def.Producer
	user     def.User
	psync    def.PSync
}

func NewMolasses(p def.Producer, u def.User, s def.PSync) *Molasses {
	return &Molasses{name: "Molasses", producer: p, user: u, psync: s}
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

func (m *Molasses) PSyncer() def.PSync {
	return m.psync
}

func (m *Molasses) SetProducer(p def.Producer) {
	m.producer = p
}

func (m *Molasses) SetUser(u def.User) {
	m.user = u
}

func (m *Molasses) SetPSyncer(s def.PSync) {
	m.psync = s
}

func (m *Molasses) TryProposeTrx() {
	if m.producer != nil {
		m.producer.TryPropose()
	}
}

func (m *Molasses) TryProposePSync() {
	if m.psync != nil {
		m.psync.TryPropose()
	}
}

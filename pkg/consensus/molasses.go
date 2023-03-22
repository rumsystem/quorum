package consensus

import (
	"github.com/rumsystem/quorum/pkg/consensus/def"
)

type Molasses struct {
	name     string
	producer def.Producer
	user     def.User
}

func NewMolasses(p def.Producer, u def.User) *Molasses {
	return &Molasses{name: "Molasses", producer: p, user: u}
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

func (m *Molasses) SetProducer(p def.Producer) {
	m.producer = p
}

func (m *Molasses) SetUser(u def.User) {
	m.user = u
}

func (m *Molasses) StartPropose() {
	if m.producer != nil {
		m.producer.StartPropose()
	}
}

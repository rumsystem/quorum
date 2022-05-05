package consensus

import (
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/pkg/consensus/def"
)

type Molasses struct {
	name       string
	producer   def.Producer
	user       def.User
	sssender   chaindef.SnapshotSender
	ssreceiver chaindef.SnapshotReceiver
}

func NewMolasses(p def.Producer, u def.User, sss chaindef.SnapshotSender, ssr chaindef.SnapshotReceiver) *Molasses {
	return &Molasses{name: "Molasses", producer: p, user: u, sssender: sss, ssreceiver: ssr}
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

func (m *Molasses) SnapshotSender() chaindef.SnapshotSender {
	return m.sssender
}

func (m *Molasses) SnapshotReceiver() chaindef.SnapshotReceiver {
	return m.ssreceiver
}

func (m *Molasses) SetProducer(p def.Producer) {
	m.producer = p
}

func (m *Molasses) SetUser(u def.User) {
	m.user = u
}

func (m *Molasses) SetSnapshotSender(sss chaindef.SnapshotSender) {
	m.sssender = sss
}

func (m *Molasses) SetSnapshotReceiver(ssr chaindef.SnapshotReceiver) {
	m.ssreceiver = ssr
}

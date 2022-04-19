package chain

//"fmt"

type Molasses struct {
	name       string
	producer   Producer
	user       User
	sssender   SnapshotSender
	ssreceiver SnapshotReceiver
}

func NewMolasses(p Producer, u User, sss SnapshotSender, ssr SnapshotReceiver) *Molasses {
	return &Molasses{name: "Molasses", producer: p, user: u, sssender: sss, ssreceiver: ssr}
}

func (m *Molasses) Name() string {
	return m.name
}

func (m *Molasses) Producer() Producer {
	return m.producer
}

func (m *Molasses) User() User {
	return m.user
}

func (m *Molasses) SnapshotSender() SnapshotSender {
	return m.sssender
}

func (m *Molasses) SnapshotReceiver() SnapshotReceiver {
	return m.ssreceiver
}

func (m *Molasses) SetProducer(p Producer) {
	m.producer = p
}

func (m *Molasses) SetUser(u User) {
	m.user = u
}

func (m *Molasses) SetSnapshotSender(sss SnapshotSender) {
	m.sssender = sss
}

func (m *Molasses) SetSnapshotReceiver(ssr SnapshotReceiver) {
	m.ssreceiver = ssr
}

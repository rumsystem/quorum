package chain

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	Snapshot() Snapshot
	SetProducer(p Producer)
	SetUser(u User)
	SetSnapshot(s Snapshot)
}

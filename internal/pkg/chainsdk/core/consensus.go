package chain

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	SnapshotSender() SnapshotSender
	SnapshotReceiver() SnapshotReceiver
	SetProducer(p Producer)
	SetUser(u User)
	SetSnapshotSender(sss SnapshotSender)
	SetSnapshotReceiver(ssr SnapshotReceiver)
}

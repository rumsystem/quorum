package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	SetProducer(p Producer)
	SetUser(u User)
	TryPropose()
	//SnapshotSender() chaindef.SnapshotSender
	//SnapshotReceiver() chaindef.SnapshotReceiver
	//SetSnapshotSender(sss chaindef.SnapshotSender)
	//SetSnapshotReceiver(ssr chaindef.SnapshotReceiver)
}

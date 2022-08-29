package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	SetProducer(p Producer)
	SetUser(u User)
	//SnapshotSender() chaindef.SnapshotSender
	//SnapshotReceiver() chaindef.SnapshotReceiver
	//SetSnapshotSender(sss chaindef.SnapshotSender)
	//SetSnapshotReceiver(ssr chaindef.SnapshotReceiver)
}

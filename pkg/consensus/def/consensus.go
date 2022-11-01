package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	PSyncer() PSync
	SetProducer(p Producer)
	SetUser(u User)
	SetPSyncer(s PSync)
	TryProposeTrx()
	TryProposePSync()
}

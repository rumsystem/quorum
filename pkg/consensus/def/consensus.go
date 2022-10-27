package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	PSyncer() PSyncer
	SetProducer(p Producer)
	SetUser(u User)
	SetPSyncer(s PSyncer)
	TryProposeTrx()
	TryProposePSync()
}

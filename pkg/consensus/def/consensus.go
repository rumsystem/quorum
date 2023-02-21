package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	PSync() PSync
	SetProducer(p Producer)
	SetUser(u User)
	SetPSync(s PSync)
	StartPropose()
}

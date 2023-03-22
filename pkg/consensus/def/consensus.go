package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	SetProducer(p Producer)
	SetUser(u User)
	StartPropose()
}

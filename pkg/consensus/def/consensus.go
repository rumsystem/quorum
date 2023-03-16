package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	ProducerProposer() ProducerProposer
	SetProducer(p Producer)
	SetUser(u User)
	SetProducerProposer(pp ProducerProposer)
	StartProposeTrx()
	StopProposeTrx()
}

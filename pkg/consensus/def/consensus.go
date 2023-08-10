package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
	ConsensusProposer() ConsensusProposer
	SetProducer(p Producer)
	SetUser(u User)
	SetConsensusProposer(pp ConsensusProposer)
	StartProposeTrx()
	StopProposeTrx()
}

type ConsensusRumLite interface {
	Name() string
	Producer() ProducerRumLite
	User() UserRumLite
	SetProducer(p ProducerRumLite)
	SetUser(u UserRumLite)
	StartProposeTrx()
	StopProposeTrx()
}

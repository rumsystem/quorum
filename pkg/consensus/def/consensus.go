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

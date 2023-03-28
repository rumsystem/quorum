package def

type Consensus interface {
	Name() string
	Producer() Producer
	User() User
<<<<<<< HEAD
	ConsensusProposer() ConsensusProposer
	SetProducer(p Producer)
	SetUser(u User)
	SetConsensusProposer(pp ConsensusProposer)
	StartProposeTrx()
	StopProposeTrx()
=======
	SetProducer(p Producer)
	SetUser(u User)
	StartPropose()
>>>>>>> consensus_2_main
}

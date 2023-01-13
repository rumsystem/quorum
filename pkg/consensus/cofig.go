package consensus

type Config struct {
	N         int      // participating nodes
	f         int      // faulty nodes
	Nodes     []string // pubkey list for all partticipating nodes
	BatchSize int      // maximum number of trxs will be commited in one epoch
	MyPubkey  string   // my pubkey
}

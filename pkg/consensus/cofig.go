package consensus

type Config struct {
	N            int      // participating nodes
	f            int      // faulty nodes
	Nodes        []string // all partticipating nodes
	BatchSize    int      // maximum number of trxs will be commited in one epoch
	MySignPubkey string
}

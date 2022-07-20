package hbbft

type Config struct {
	N         int      // participating nodes
	F         int      // faulty nodes
	MyNodeId  string   // my nodeid
	Nodes     []string // all partticipating nodes (include myself)
	BatchSize int      // maximum number of trxs will be commited in one epoch
}

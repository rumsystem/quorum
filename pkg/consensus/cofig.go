package consensus

type Config struct {
	GroupId     string   // group id
	NodeName    string   // node name
	MyPubkey    string   // my pubkey
	OwnerPubKey string   // owner pubkey
	N           int      // participating producer nodes
	f           int      // faulty nodes
	Nodes       []string // pubkey list for all partticipating nodes
	BatchSize   int      // maximum number of trxs will be commited in one epoch
}

package consensus

type BftStatus uint

const (
	IDLE BftStatus = iota
	RUNNING
	CLOSED
)

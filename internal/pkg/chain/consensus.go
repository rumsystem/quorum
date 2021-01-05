package chain

type Consensus interface {
	Name() string
	Producer() Producer
}

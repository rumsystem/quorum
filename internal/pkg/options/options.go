package options

import (
	logging "github.com/ipfs/go-log/v2"
	"sync"
)

var optionslog = logging.Logger("options")

type NodeOptions struct {
	EnableNat         bool
	EnableRumExchange bool
	EnableDevNetwork  bool
	IsRexTestMode     bool
	MaxPeers          int
	ConnsHi           int
	NetworkName       string
	JWTToken          string
	JWTKey            string
	SignKeyMap        map[string]string
	mu                sync.RWMutex
}

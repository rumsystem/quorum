package options

import (
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
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

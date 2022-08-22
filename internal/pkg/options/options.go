package options

import (
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var optionslog = logging.Logger("options")

type NodeOptions struct {
	Password          string
	EnableRelay       bool
	EnableNat         bool
	EnableRumExchange bool
	EnableDevNetwork  bool
	EnableSnapshot    bool
	IsRexTestMode     bool
	MaxPeers          int
	ConnsHi           int
	NetworkName       string
	JWTTokenMap       map[string]string
	JWTKey            string
	SignKeyMap        map[string]string
	mu                sync.RWMutex
}

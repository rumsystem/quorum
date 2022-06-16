package options

import (
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var optionslog = logging.Logger("options")

type NodeOptions struct {
	EnableRelay        bool
	EnableRelayService bool /* this will force the node to be public */
	EnableNat          bool
	EnableRumExchange  bool
	EnableDevNetwork   bool
	IsRexTestMode      bool
	MaxPeers           int
	ConnsHi            int
	NetworkName        string
	JWTTokenMap        map[string]string
	JWTKey             string
	SignKeyMap         map[string]string
	mu                 sync.RWMutex
}

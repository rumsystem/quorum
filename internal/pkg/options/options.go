package options

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"sync"
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
	SelfJWTToken       string
	OthersJWTToken     string
	JWTKey             string
	SignKeyMap         map[string]string
	mu                 sync.RWMutex
}

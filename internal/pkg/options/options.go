package options

import (
	"sync"

	logging "github.com/ipfs/go-log/v2"
)

var optionslog = logging.Logger("options")

type NodeOptions struct {
	EnableNat        bool
	EnableDevNetwork bool
	NetworkName      string
	JWTToken         string
	JWTKey           string
	SignKeyMap       map[string]string
	mu               sync.RWMutex
}

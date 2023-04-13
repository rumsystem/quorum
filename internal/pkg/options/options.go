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
	EnablePubQue      bool
	MaxPeers          int
	ConnsHi           int
	NetworkName       string
	JWT               *JWT
	SignKeyMap        map[string]string
	mu                sync.RWMutex
}

type (
	JWT struct {
		Key   string                  `json:"key" mapstructure:"key"`
		Chain *JWTListItem            `json:"chain" mapstructure:"chain"`
		Node  map[string]*JWTListItem `json:"node" mapstructure:"node"`
	}

	JWTListItem struct {
		Normal []*TokenItem `json:"normal" mapstructure:"normal"`
		Revoke []*TokenItem `json:"revoke" mapstructure:"revoke"`
	}

	TokenItem struct {
		Remark string `json:"remark" mapstructure:"remark"`
		Token  string `json:"token" mapstructure:"token"`
	}
)

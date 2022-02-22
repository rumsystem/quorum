package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func MgrChainConfig(data []byte) (*handlers.ChainConfigResult, error) {
	params := &handlers.ChainConfigParams{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.MgrChainConfig(params)
}

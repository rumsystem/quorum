//go:build js && wasm
// +build js,wasm

package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func MgrGroupConfig(data []byte) (*handlers.GroupConfigResult, error) {
	params := &handlers.GroupConfigParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.MgrGroupConfig(params)
}

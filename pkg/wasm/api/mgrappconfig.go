//go:build js && wasm
// +build js,wasm

package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/chainsdk/handlers"
)

func MgrAppConfig(data []byte) (*handlers.AppConfigResult, error) {
	params := &handlers.AppConfigParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.MgrAppConfig(params)
}

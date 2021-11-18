//go:build js && wasm
// +build js,wasm

package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func CreateGroup(data []byte) (*handlers.CreateGroupResult, error) {
	wasmCtx := quorumContext.GetWASMContext()

	params := &handlers.CreateGroupParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.CreateGroup(params, wasmCtx.NodeOpt)
}

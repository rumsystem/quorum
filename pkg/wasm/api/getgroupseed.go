//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func GetGroupSeed(groupId string) (*handlers.GroupSeed, error) {
	wasmCtx := quorumContext.GetWASMContext()

	return handlers.GetGroupSeed(groupId, wasmCtx.AppDb)
}

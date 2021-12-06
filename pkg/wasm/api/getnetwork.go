//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func GetNetwork() (*handlers.NetworkInfo, error) {
	wasmCtx := quorumContext.GetWASMContext()

	return handlers.GetNetwork(&wasmCtx.QNode.Host, wasmCtx.QNode.Info, wasmCtx.NodeOpt, wasmCtx.EthAddr)
}

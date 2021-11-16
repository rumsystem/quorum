//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func GetNodeInfo() (*handlers.NodeInfo, error) {
	wasmCtx := quorumContext.GetWASMContext()

	return handlers.GetNodeInfo(wasmCtx.QNode.NetworkName)
}

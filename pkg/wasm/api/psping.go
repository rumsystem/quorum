package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func Ping(peer string) (*handlers.PingResp, error) {
	wasmCtx := quorumContext.GetWASMContext()

	return handlers.Ping(wasmCtx.QNode.Pubsub, wasmCtx.QNode.Host.ID(), peer)
}

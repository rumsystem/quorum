package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func Ping(peer string) (*handlers.PingResp, error) {
	wasmCtx := quorumContext.GetWASMContext()

	return handlers.Ping(wasmCtx.QNode.Pubsub, wasmCtx.QNode.Host.ID(), peer)
}

func P2PPing(peer string) (*handlers.P2PPingResp, error) {
	wasmCtx := quorumContext.GetWASMContext()

	return handlers.P2PPing(wasmCtx.QNode.Host, peer)
}

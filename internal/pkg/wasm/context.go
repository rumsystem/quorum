// +build js,wasm

package wasm

import (
	"context"

	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
)

type QuorumWasmContext struct {
	QNode  *quorumP2P.Node
	Config *BrowserConfig

	Ctx    context.Context
	Cancel context.CancelFunc

	Qchan chan struct{}
}

func NewQuorumWasmContext(qchan chan struct{}, config *BrowserConfig, node *quorumP2P.Node, ctx context.Context, cancel context.CancelFunc) *QuorumWasmContext {
	qCtx := QuorumWasmContext{node, config, ctx, cancel, qchan}
	return &qCtx
}

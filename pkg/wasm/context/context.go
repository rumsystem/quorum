//go:build js && wasm
// +build js,wasm

package context

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumConfig "github.com/rumsystem/quorum/pkg/wasm/config"
)

/* global, JS should interact with it */
var wasmCtx *QuorumWasmContext = nil

func GetWASMContext() *QuorumWasmContext {
	return wasmCtx
}

func Init(qchan chan struct{}, config *quorumConfig.BrowserConfig, node *quorumP2P.Node, appDb *appdata.AppDb, dbMgr *storage.DbMgr, ctx context.Context, cancel context.CancelFunc) {
	wasmCtx = NewQuorumWasmContext(qchan, config, node, appDb, dbMgr, ctx, cancel)
}

type QuorumWasmContext struct {
	QNode  *quorumP2P.Node
	AppDb  *appdata.AppDb
	DbMgr  *storage.DbMgr
	Config *quorumConfig.BrowserConfig

	Ctx    context.Context
	Cancel context.CancelFunc

	Qchan chan struct{}
}

func NewQuorumWasmContext(qchan chan struct{}, config *quorumConfig.BrowserConfig, node *quorumP2P.Node, appDb *appdata.AppDb, dbMgr *storage.DbMgr, ctx context.Context, cancel context.CancelFunc) *QuorumWasmContext {
	qCtx := QuorumWasmContext{node, appDb, dbMgr, config, ctx, cancel, qchan}
	return &qCtx
}

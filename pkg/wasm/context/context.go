//go:build js && wasm
// +build js,wasm

package context

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumConfig "github.com/rumsystem/quorum/pkg/wasm/config"
)

/* global, JS should interact with it */
var wasmCtx *QuorumWasmContext = nil

func GetWASMContext() *QuorumWasmContext {
	return wasmCtx
}

func Init(qchan chan struct{}, config *quorumConfig.BrowserConfig, node *quorumP2P.Node, ethAddr string, nodeOpt *options.NodeOptions, appDb *appdata.AppDb, dbMgr *storage.DbMgr, ctx context.Context, cancel context.CancelFunc) {
	wasmCtx = NewQuorumWasmContext(qchan, config, node, ethAddr, nodeOpt, appDb, dbMgr, ctx, cancel)
}

type QuorumWasmContext struct {
	EthAddr string
	NodeOpt *options.NodeOptions
	QNode   *quorumP2P.Node
	AppDb   *appdata.AppDb
	DbMgr   *storage.DbMgr
	Config  *quorumConfig.BrowserConfig

	Ctx    context.Context
	Cancel context.CancelFunc

	Qchan        chan struct{}
	PubqueueChan chan bool
}

func NewQuorumWasmContext(qchan chan struct{}, config *quorumConfig.BrowserConfig, node *quorumP2P.Node, ethAddr string, nodeOpt *options.NodeOptions, appDb *appdata.AppDb, dbMgr *storage.DbMgr, ctx context.Context, cancel context.CancelFunc) *QuorumWasmContext {
	pubqueueChan := make(chan bool)
	qCtx := QuorumWasmContext{ethAddr, nodeOpt, node, appDb, dbMgr, config, ctx, cancel, qchan, pubqueueChan}
	return &qCtx
}

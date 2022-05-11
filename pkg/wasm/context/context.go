//go:build js && wasm
// +build js,wasm

package context

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	quorumConfig "github.com/rumsystem/quorum/pkg/wasm/config"
)

/* global, JS should interact with it */
var wasmCtx *QuorumWasmContext = nil

func GetWASMContext() *QuorumWasmContext {
	return wasmCtx
}

func Init(qchan chan struct{}, config *quorumConfig.BrowserConfig, node *quorumP2P.Node, ethAddr string, nodeOpt *options.NodeOptions, appDb *appdata.AppDb, chaindb *chainstorage.Storage, dbMgr *storage.DbMgr, ctx context.Context, cancel context.CancelFunc) {
	wasmCtx = NewQuorumWasmContext(qchan, config, node, ethAddr, nodeOpt, appDb, chaindb, dbMgr, ctx, cancel)
}

type QuorumWasmContext struct {
	EthAddr string
	NodeOpt *options.NodeOptions
	QNode   *quorumP2P.Node
	AppDb   *appdata.AppDb
	chaindb *chainstorage.Storage
	DbMgr   *storage.DbMgr
	Config  *quorumConfig.BrowserConfig

	Ctx    context.Context
	Cancel context.CancelFunc

	Qchan        chan struct{}
	PubqueueChan chan bool
}

func (wasmCtx *QuorumWasmContext) GetChainStorage() *chainstorage.Storage {
	return wasmCtx.chaindb
}

func NewQuorumWasmContext(qchan chan struct{}, config *quorumConfig.BrowserConfig, node *quorumP2P.Node, ethAddr string, nodeOpt *options.NodeOptions, appDb *appdata.AppDb, chaindb *chainstorage.Storage, dbMgr *storage.DbMgr, ctx context.Context, cancel context.CancelFunc) *QuorumWasmContext {
	pubqueueChan := make(chan bool)
	qCtx := QuorumWasmContext{ethAddr, nodeOpt, node, appDb, chaindb, dbMgr, config, ctx, cancel, qchan, pubqueueChan}
	return &qCtx
}

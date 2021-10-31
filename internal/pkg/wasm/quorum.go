//go:build js && wasm
// +build js,wasm

package wasm

import (
	"context"
	"crypto/rand"

	ethKeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
)

/* global, JS should interact with it */
var wasmCtx *QuorumWasmContext = nil

func StartQuorum(qchan chan struct{}, bootAddrsStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	config := NewBrowserConfig([]string{bootAddrsStr})

	nodeOpt := options.NodeOptions{}
	nodeOpt.EnableNat = false
	nodeOpt.NetworkName = config.NetworkName
	nodeOpt.EnableDevNetwork = config.UseTestNet

	dbMgr, err := newStoreManager()
	if err != nil {
		panic(err)
	}

	/* Randomly genrate a key
	TODO: should load from somewhere(IndexedDB or user localfile etc.) */
	key := ethKeystore.NewKeyForDirectICAP(rand.Reader)

	node, err := quorumP2P.NewBrowserNode(ctx, &nodeOpt, key)
	if err != nil {
		panic(nil)
	}

	nodectx.InitCtx(ctx, "default", node, dbMgr, "pubsub", "wasm-version")
	// TODO: init keystore
	// ksi: keystore load from local by calling `InitDirKeyStore`
	//nodectx.GetNodeCtx().Keystore = ksi
	//nodectx.GetNodeCtx().PublicKey = keys.PubKey
	//nodectx.GetNodeCtx().PeerId = peerid

	/* quorum has global groupmgr, init it here */
	groupmgr := chain.InitGroupMgr(dbMgr)

	// TODO: construct app db

	wasmCtx = NewQuorumWasmContext(qchan, config, node, ctx, cancel)

	/* Bootstrap will connect to all bootstrap nodes in config.
	since we can not listen in browser, there is no need to anounce */
	wasmCtx.Bootstrap()

	/* TODO: should also try to connect known peers in peerstore which is
	   not implemented yet */

	/* keep finding peers, and try to connect to them */
	go wasmCtx.StartDiscoverTask()

	/* start syncing all local groups */
	err = groupmgr.SyncAllGroup()
	if err != nil {
		panic(err)
	}
}

func newStoreManager() (*storage.DbMgr, error) {
	groupDb := quorumStorage.QSIndexDB{}
	err := groupDb.Init("quorum_groups")
	if err != nil {
		return nil, err
	}
	dataDb := quorumStorage.QSIndexDB{}
	err = dataDb.Init("quorum_data")
	if err != nil {
		return nil, err
	}

	storeMgr := storage.DbMgr{}
	storeMgr.GroupInfoDb = &groupDb
	storeMgr.Db = &dataDb
	storeMgr.Auth = nil
	storeMgr.DataPath = "."

	return &storeMgr, nil
}

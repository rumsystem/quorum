//go:build js && wasm
// +build js,wasm

package wasm

import (
	"context"
	"errors"

	ethKeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	quorumCrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
)

const DEFAUT_KEY_NAME string = "default"

/* global, JS should interact with it */
var wasmCtx *QuorumWasmContext = nil

func StartQuorum(qchan chan struct{}, bootAddrsStr string) (bool, error) {
	ctx, cancel := context.WithCancel(context.Background())

	config := NewBrowserConfig([]string{bootAddrsStr})

	nodeOpt := options.NodeOptions{}
	nodeOpt.EnableNat = false
	nodeOpt.NetworkName = config.NetworkName
	nodeOpt.EnableDevNetwork = config.UseTestNet

	dbMgr, err := newStoreManager()
	if err != nil {
		cancel()
		return false, err
	}

	// TODO: read from user
	password := "password"

	/* init browser keystore */
	k, err := quorumCrypto.InitBrowserKeystore(password)
	if err != nil {
		cancel()
		return false, err
	}
	ks := k.(*quorumCrypto.BrowserKeystore)
	println("InitBrowserKeystore OK")

	/* get default sign key */
	key, err := ks.GetUnlockedKey(quorumCrypto.Sign.NameString(DEFAUT_KEY_NAME))
	if err != nil {
		cancel()
		return false, err
	}

	defaultKey, ok := key.(*ethKeystore.Key)
	if !ok {
		cancel()
		return false, errors.New("failed to cast key")
	}
	println("defaultKey OK")

	node, err := quorumP2P.NewBrowserNode(ctx, &nodeOpt, defaultKey)
	if err != nil {
		cancel()
		return false, err
	}

	nodectx.InitCtx(ctx, "default", node, dbMgr, "pubsub", "wasm-version")
	nodectx.GetNodeCtx().Keystore = k

	keys, err := quorumCrypto.SignKeytoPeerKeys(defaultKey)
	if err != nil {
		cancel()
		return false, err
	}
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	println("SignKeytoPeerKeys OK")

	peerId, _, err := ks.GetPeerInfo(DEFAUT_KEY_NAME)
	if err != nil {
		cancel()
		return false, err
	}
	nodectx.GetNodeCtx().PeerId = peerId

	println("GetPeerInfo OK")
	/* quorum has global groupmgr, init it here */
	groupmgr := chain.InitGroupMgr(dbMgr)

	println("InitGroupMgr OK")
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
		return false, err
	}

	return true, nil
}

func newStoreManager() (*storage.DbMgr, error) {
	groupDb := quorumStorage.QSIndexDB{}
	err := groupDb.Init("groups")
	if err != nil {
		return nil, err
	}
	dataDb := quorumStorage.QSIndexDB{}
	err = dataDb.Init("data")
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

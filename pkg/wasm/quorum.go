//go:build js && wasm
// +build js,wasm

package wasm

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	ethKeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	quorumCrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumConfig "github.com/rumsystem/quorum/pkg/wasm/config"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
	"github.com/rumsystem/quorum/pkg/wasm/logger"
)

const DEFAUT_KEY_NAME string = "default"

func StartQuorum(qchan chan struct{}, password string, bootAddrs []string) (bool, error) {
	ctx, cancel := context.WithCancel(context.Background())

	config := quorumConfig.NewBrowserConfig(bootAddrs)

	nodeOpt := options.NodeOptions{}
	nodeOpt.EnableNat = false
	nodeOpt.NetworkName = config.NetworkName
	nodeOpt.EnableDevNetwork = config.UseTestNet

	dbMgr, err := newStoreManager()
	if err != nil {
		cancel()
		return false, err
	}

	/* init browser keystore */
	k, err := quorumCrypto.InitBrowserKeystore(password)
	if err != nil {
		cancel()
		return false, err
	}
	ks := k.(*quorumCrypto.BrowserKeystore)
	logger.Console.Log("InitBrowserKeystore OK")

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
	logger.Console.Log("defaultKey OK")

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
	logger.Console.Log("SignKeytoPeerKeys OK")

	peerId, ethAddr, err := ks.GetPeerInfo(DEFAUT_KEY_NAME)
	if err != nil {
		cancel()
		return false, err
	}
	nodectx.GetNodeCtx().PeerId = peerId
	logger.Console.Log("GetPeerInfo OK")

	chain.InitGroupMgr(dbMgr)
	logger.Console.Log("InitGroupMgr OK")

	appIndexedDb, err := newAppDb()
	if err != nil {
		cancel()
		return false, err
	}
	appDb := appdata.NewAppDb()
	appDb.Db = appIndexedDb

	quorumContext.Init(qchan, config, node, ethAddr, &nodeOpt, appDb, dbMgr, ctx, cancel)

	storage.InitSeqenceDB()

	/* Bootstrap will connect to all bootstrap nodes in config.
	since we can not listen in browser, there is no need to anounce */
	err = Bootstrap()
	if err != nil {
		return false, err
	}

	return true, nil
}

func newAppDb() (*storage.QSIndexDB, error) {
	appDb := quorumStorage.QSIndexDB{}
	err := appDb.Init("app")
	if err != nil {
		return nil, err
	}
	return &appDb, nil
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

func Bootstrap() error {
	wasmCtx := quorumContext.GetWASMContext()

	/* connect to bootstraps */
	bootstraps := []peer.AddrInfo{}
	for _, peerAddr := range wasmCtx.Config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		bootstraps = append(bootstraps, *peerinfo)
	}
	connectedPeers := wasmCtx.QNode.AddPeers(wasmCtx.Ctx, bootstraps)
	logger.Console.Log(fmt.Sprintf("Connected to %d peers", connectedPeers))

	/* keep finding peers, and try to connect to them */
	go StartDiscoverTask()

	/* start syncing all local groups */
	groupMgr := chain.GetGroupMgr()
	err := groupMgr.SyncAllGroup()
	if err != nil {
		return err
	}
	logger.Console.Log("Group Syncer Started")

	/* new syncer for app data */
	appsync := appdata.NewAppSyncAgent("", "default", wasmCtx.AppDb, wasmCtx.DbMgr)
	appsync.Start(10)
	logger.Console.Log("App Syncer Started")

	return nil
}

func StartDiscoverTask() {
	wasmCtx := quorumContext.GetWASMContext()
	var doDiscoverTask = func() {
		logger.Console.Log("Searching for other peers...")
		peerChan, err := wasmCtx.QNode.RoutingDiscovery.FindPeers(wasmCtx.Ctx, quorumConfig.DefaultRendezvousString)
		if err != nil {
			logger.Console.Error(err.Error())
			return
		}

		var connectCount uint32 = 0

		for peer := range peerChan {
			curPeer := peer
			logger.Console.Log("Found peer:" + curPeer.String())
			go func() {
				pctx, cancel := context.WithTimeout(wasmCtx.Ctx, time.Second*10)
				defer cancel()
				err := wasmCtx.QNode.Host.Connect(pctx, curPeer)
				if err != nil {
					logger.Console.Error("Failed to connect peer: " + curPeer.String())
				} else {
					curConnectedCount := atomic.AddUint32(&connectCount, 1)
					logger.Console.Log(fmt.Sprintf("Connected to peer(%d): %s", curConnectedCount, curPeer.String()))
				}
			}()
		}
	}
	/* first job will start after 1 second */
	go func() {
		time.Sleep(1 * time.Second)
		doDiscoverTask()
	}()

	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			go doDiscoverTask()
		case <-wasmCtx.Qchan:
			ticker.Stop()
			wasmCtx.Cancel()
		}
	}
}

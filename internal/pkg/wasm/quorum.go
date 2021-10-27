// +build js,wasm

package wasm

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	ethKeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
)

func StartQuorum(qchan chan struct{}, bootAddrsStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	config := NewBrowserConfig([]string{bootAddrsStr})

	nodeOpt := options.NodeOptions{}
	nodeOpt.EnableNat = false
	nodeOpt.NetworkName = config.NetworkName
	nodeOpt.EnableDevNetwork = config.UseTestNet

	/* Randomly genrate a key
	TODO: should load from somewhere(IndexedDB or user localfile etc.) */
	key := ethKeystore.NewKeyForDirectICAP(rand.Reader)

	node, err := quorumP2P.NewBrowserNode(ctx, &nodeOpt, key)
	if err != nil {
		panic(nil)
	}

	wasmCtx := NewQuorumWasmContext(qchan, config, node, ctx, cancel)

	/* Bootstrap will connect to all bootstrap nodes in config.
	since we can not listen in browser, there is no need to anounce */
	Bootstrap(wasmCtx)

	/* TODO: should also try to connect known peers in peerstore which is
	   not implemented yet */

	go startBackgroundWork(wasmCtx)
}

func Bootstrap(wasmCtx *QuorumWasmContext) {
	bootstraps := []peer.AddrInfo{}
	for _, peerAddr := range wasmCtx.Config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		bootstraps = append(bootstraps, *peerinfo)
	}

	connectedPeers := wasmCtx.QNode.AddPeers(wasmCtx.Ctx, bootstraps)
	println(fmt.Sprintf("Connected to %d peers", connectedPeers))
}

func startBackgroundWork(wasmCtx *QuorumWasmContext) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			// Now, look for others who have announced
			// This is like your friend telling you the location to meet you.
			println("Searching for other peers...")
			peerChan, err := wasmCtx.QNode.RoutingDiscovery.FindPeers(wasmCtx.Ctx, DefaultRendezvousString)
			if err != nil {
				panic(err)
			}

			for peer := range peerChan {
				if peer.ID == wasmCtx.QNode.Host.ID() {
					// println("Found peer(self):", peer.String())
				} else {
					println("Found peer:", peer.String())
				}
			}
		case <-wasmCtx.Qchan:
			ticker.Stop()
			wasmCtx.Cancel()
		}
	}
}

//go:build js && wasm
// +build js,wasm

package wasm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
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

func (wasmCtx *QuorumWasmContext) Bootstrap() {
	bootstraps := []peer.AddrInfo{}
	for _, peerAddr := range wasmCtx.Config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		bootstraps = append(bootstraps, *peerinfo)
	}

	connectedPeers := wasmCtx.QNode.AddPeers(wasmCtx.Ctx, bootstraps)
	println(fmt.Sprintf("Connected to %d peers", connectedPeers))
}

func (wasmCtx *QuorumWasmContext) StartDiscoverTask() {
	var doDiscoverTask = func() {
		println("Searching for other peers...")
		peerChan, err := wasmCtx.QNode.RoutingDiscovery.FindPeers(wasmCtx.Ctx, DefaultRendezvousString)
		if err != nil {
			panic(err)
		}

		var connectCount uint32 = 0

		for peer := range peerChan {
			curPeer := peer
			println("Found peer:", curPeer.String())
			go func() {
				pctx, cancel := context.WithTimeout(wasmCtx.Ctx, time.Second*10)
				defer cancel()
				err := wasmCtx.QNode.Host.Connect(pctx, curPeer)
				if err != nil {
					println("Failed to connect peer: ", curPeer.String())
					cancel()
				} else {
					curConnectedCount := atomic.AddUint32(&connectCount, 1)
					println(fmt.Sprintf("Connected to peer(%d): %s", curConnectedCount, curPeer.String()))
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
			doDiscoverTask()
		case <-wasmCtx.Qchan:
			ticker.Stop()
			wasmCtx.Cancel()
		}
	}
}

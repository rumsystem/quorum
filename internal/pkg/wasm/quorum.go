// go:build js && wasm
// +build js,wasm
package wasm

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	ethKeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/google/orderedcode"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumP2P "github.com/rumsystem/quorum/internal/pkg/p2p"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
)

type QuorumWasmContext struct {
	QNode *quorumP2P.Node

	Ctx    context.Context
	Cancel context.CancelFunc

	Qchan chan struct{}
}

func NewQuorumWasmContext(qchan chan struct{}, node *quorumP2P.Node, ctx context.Context, cancel context.CancelFunc) *QuorumWasmContext {
	qCtx := QuorumWasmContext{node, ctx, cancel, qchan}
	return &qCtx
}

func StringsToAddrs(addrStrings []string) (maddrs []ma.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := ma.NewMultiaddr(addrString)
		if err != nil {
			println(err.Error())
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

// TODO: read from config
var DefaultRendezvousString = "e6629921-b5cd-4855-9fcd-08bcc39caef7"
var DefaultRoutingProtoPrefix = "/quorum/nevis"
var DefaultNetworkName = "nevis"
var DefaultPubsubProtocol = "/quorum/nevis/meshsub/1.1.0"

func StartQuorum(qchan chan struct{}, bootAddrsStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	bootAddrs, _ := StringsToAddrs([]string{bootAddrsStr})

	// TODO: load options and key
	nodeOpt := options.NodeOptions{}
	nodeOpt.EnableDevNetwork = false
	nodeOpt.EnableNat = false
	nodeOpt.NetworkName = DefaultNetworkName

	key := ethKeystore.NewKeyForDirectICAP(rand.Reader)

	node, err := quorumP2P.NewBrowserNode(ctx, &nodeOpt, key)
	if err != nil {
		panic(nil)
	}

	// TODO: connect to boot address from peerstore
	peers := []peer.AddrInfo{}
	for _, peerAddr := range bootAddrs {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		peers = append(peers, *peerinfo)
	}

	connectedPeers := node.AddPeers(ctx, peers)
	println(fmt.Sprintf("Connected to %d peers", connectedPeers))

	// annouce self
	println("Announcing ourselves...")
	discovery.Advertise(ctx, node.RoutingDiscovery, DefaultRendezvousString)
	println("Successfully announced!")

	wasmCtx := NewQuorumWasmContext(qchan, node, ctx, cancel)
	go startBackgroundWork(wasmCtx)
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

func IndexDBTest() {
	dbMgr := quorumStorage.QSIndexDB{}
	err := dbMgr.Init("test")
	if err != nil {
		panic(err)
	}

	{
		k := []byte("key")
		v := []byte("value")
		err = dbMgr.Set(k, v)
		if err != nil {
			panic(err)
		}

		val, err := dbMgr.Get(k)
		if err != nil {
			panic(err)
		}
		println(string(k), string(val))

		err = dbMgr.Delete(k)
		if err != nil {
			panic(err)
		}

		val, err = dbMgr.Get(k)
		if err != nil {
			println("not found (OK)")
		}
	}
	{
		keys := []string{}
		values := []string{}
		keyPrefix := "key"
		i := 0
		for i < 100 {
			k, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(i))
			keys = append(keys, string(k))
			values = append(values, fmt.Sprintf("value-%d", i))
			i += 1
		}
		for idx, k := range keys {
			err = dbMgr.Set([]byte(k), []byte(values[idx]))
			if err != nil {
				panic(err)
			}
		}
		rKey, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(100))
		i = 0
		err = dbMgr.PrefixForeachKey(rKey, []byte(keyPrefix), true, func(k []byte, err error) error {
			i += 1
			curKey, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(100-i))
			if !bytes.Equal(k, curKey) {
				return errors.New("wrong key")
			}
			return nil
		})
		if err != nil {
			panic(err)
		}

		i = 0
		err = dbMgr.PrefixForeachKey([]byte(keyPrefix), []byte(keyPrefix), false, func(k []byte, err error) error {
			curKey, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(i))
			i += 1
			if !bytes.Equal(k, curKey) {
				return errors.New("wrong key")
			}
			return nil
		})
		if err != nil {
			panic(err)
		}

		for _, k := range keys {
			err = dbMgr.Delete([]byte(k))
			if err != nil {
				panic(err)
			}
		}

		println("Test Done: OK")
	}

}

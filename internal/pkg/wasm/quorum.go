// go:build js && wasm
// +build js,wasm
package wasm

import (
	"context"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
)

type QuorumWasmNode struct {
	Ps               *pubsub.PubSub
	Host             host.Host
	RoutingDiscovery *discovery.RoutingDiscovery

	Ctx    context.Context
	Cancel context.CancelFunc

	Qchan chan struct{}
}

func newQuorumWasmNode(ps *pubsub.PubSub, host host.Host, routingDiscovery *discovery.RoutingDiscovery, ctx context.Context, cancel context.CancelFunc, qchan chan struct{}) *QuorumWasmNode {
	ret := QuorumWasmNode{
		ps, host, routingDiscovery, ctx, cancel, qchan,
	}

	return &ret
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

var DefaultRendezvousString = "e6629921-b5cd-4855-9fcd-08bcc39caef7"
var DefaultRoutingProtoPrefix = "/quorum/nevis"
var DefaultPubsubProtocol = "/quorum/nevis/meshsub/1.1.0"

func StartQuorum(qchan chan struct{}, bootAddrsStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	bootAddrs, _ := StringsToAddrs([]string{bootAddrsStr})

	var routingDiscovery *discovery.RoutingDiscovery
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeClient),
			dht.Concurrency(10),
			dht.ProtocolPrefix(protocol.ID(DefaultRoutingProtoPrefix)),
		)

		var err error
		ddht, err := dual.New(ctx, host, dhtOpts)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	// WebSockets only:
	h, err := libp2p.New(
		ctx,
		routing,
		libp2p.Transport(ws.New),
		libp2p.ListenAddrs(),
	)
	if err != nil {
		panic(err)
	}

	println("id: ", h.ID().String())

	psOptions := []pubsub.Option{pubsub.WithPeerExchange(true)}

	qProto := protocol.ID(DefaultPubsubProtocol)
	protos := []protocol.ID{qProto}
	features := func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		if proto == qProto {
			return true
		}
		return false
	}
	psOptions = append(psOptions, pubsub.WithGossipSubProtocols(protos, features))

	psOptions = append(psOptions, pubsub.WithPeerOutboundQueueSize(128))

	ps, err := pubsub.NewGossipSub(ctx, h, psOptions...)

	println(ps)

	for _, peerAddr := range bootAddrs {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		url := peerAddr.String()
		println("connecting: ", url)
		if err := h.Connect(ctx, *peerinfo); err != nil {
			panic(err)
		} else {
			println("Connection established with bootstrap node: ", url)

		}
	}

	println("Announcing ourselves...")
	discovery.Advertise(ctx, routingDiscovery, DefaultRendezvousString)
	println("Successfully announced!")

	node := newQuorumWasmNode(ps, h, routingDiscovery, ctx, cancel, qchan)

	go startBackgroundWork(node)
}

func startBackgroundWork(node *QuorumWasmNode) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			// Now, look for others who have announced
			// This is like your friend telling you the location to meet you.
			println("Searching for other peers...")
			peerChan, err := node.RoutingDiscovery.FindPeers(node.Ctx, DefaultRendezvousString)
			if err != nil {
				panic(err)
			}

			for peer := range peerChan {
				if peer.ID == node.Host.ID() {
					// println("Found peer(self):", peer.String())
				} else {
					println("Found peer:", peer.String())
				}
			}
		case <-node.Qchan:
			ticker.Stop()
			node.Cancel()
		}
	}
}

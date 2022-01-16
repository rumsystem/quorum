//go:build js && wasm
// +build js,wasm

package p2p

import (
	"context"
	"fmt"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ws "github.com/libp2p/go-ws-transport"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

func NewBrowserNode(ctx context.Context, nodeOpt *options.NodeOptions, key *ethkeystore.Key) (*Node, error) {
	var ddht *dual.DHT
	var routingDiscovery *discovery.RoutingDiscovery
	var err error

	//privKey p2pcrypto.PrivKey
	privKey := key.PrivateKey
	priv, err := p2pcrypto.UnmarshalSecp256k1PrivateKey(ethcrypto.FromECDSA(privKey))
	if err != nil {
		return nil, err
	}

	nodeNetwork := nodeOpt.NetworkName
	if nodeOpt.EnableDevNetwork == true {
		nodeNetwork = fmt.Sprintf("%s-%s", nodeOpt.NetworkName, "dev")
	}

	routingProtocolPrefix := fmt.Sprintf("%s/%s", ProtocolPrefix, nodeNetwork)
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeClient),
			dht.Concurrency(10),
			dht.ProtocolPrefix(protocol.ID(routingProtocolPrefix)),
		)

		var err error
		ddht, err = dual.New(ctx, host, dhtOpts)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	networklog.Infof("Enable dht protocol prefix: %s", routingProtocolPrefix)

	identity := libp2p.Identity(priv)

	host, err := libp2p.New(libp2p.ListenAddrs(),
		libp2p.Transport(ws.New),
		routing,
		libp2p.Ping(false),
		identity,
	)
	if err != nil {
		return nil, err
	}

	// configure our own ping protocol
	pingService := &PingService{Host: host}
	host.SetStreamHandler(PingID, pingService.PingHandler)
	options := []pubsub.Option{pubsub.WithPeerExchange(true)}

	networklog.Infof("Network Name %s", nodeNetwork)

	var ps *pubsub.PubSub

	// TODO: store tracer

	customProtocol := protocol.ID(fmt.Sprintf("%s/meshsub/1.1.0", fmt.Sprintf("%s/%s", ProtocolPrefix, nodeNetwork)))
	protos := []protocol.ID{customProtocol}
	features := func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		if proto == customProtocol {
			return true
		}
		return false
	}

	networklog.Infof("Enable protocol: %s", customProtocol)

	options = append(options, pubsub.WithGossipSubProtocols(protos, features))
	options = append(options, pubsub.WithPeerOutboundQueueSize(128))

	ps, err = pubsub.NewGossipSub(ctx, host, options...)

	if err != nil {
		return nil, err
	}

	psPing := NewPSPingService(ctx, ps, host.ID())
	psPing.EnablePing()
	info := &NodeInfo{NATType: network.ReachabilityUnknown}

	newNode := &Node{NetworkName: nodeNetwork, Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery, Info: info}

	// TODO: store peers and reconnect them

	go newNode.eventhandler(ctx)

	return newNode, nil
}

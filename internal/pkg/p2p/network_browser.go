//go:build js && wasm
// +build js,wasm

package p2p

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/control"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/pubsubconn"
)

type PublicGater struct{}

func (PublicGater) InterceptPeerDial(p peer.ID) (allow bool) {
	return true
}
func (PublicGater) InterceptAddrDial(_ peer.ID, addr ma.Multiaddr) (allow bool) {
	supportWs := false
	// check ws
	for _, proto := range addr.Protocols() {
		if proto.Name == "ws" {
			supportWs = true
		}
	}
	if !supportWs {
		return false
	}
	if strings.HasPrefix(addr.String(), "/ip4/127.0.0.1/") {
		return false
	}
	if strings.HasPrefix(addr.String(), "/ip4/10.") { // 10.0.0.0/8
		return false
	}
	if strings.HasPrefix(addr.String(), "/ip4/172.") {
		// 172.16.0.0/12
		tail := strings.ReplaceAll(addr.String(), "/ip4/172.", "")
		tailArr := strings.Split(tail, ".")
		n, _ := strconv.Atoi(tailArr[0])
		if n >= 16 || n <= 31 {
			return false
		}
		return true
	}
	if strings.HasPrefix(addr.String(), "/ip4/169.254.") { // 169.254.0.0/16
		return false
	}
	if strings.HasPrefix(addr.String(), "/ip4/192.168.") { // 192.168.0.0/16
		return false
	}
	return true
}
func (PublicGater) InterceptAccept(network.ConnMultiaddrs) (allow bool) {
	return true
}
func (PublicGater) InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool) {
	return true
}
func (PublicGater) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

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

	publicGater := &PublicGater{}

	host, err := libp2p.New(libp2p.ListenAddrs(),
		libp2p.Transport(ws.New),
		routing,
		libp2p.Ping(false),
		identity,
		libp2p.ConnectionGater(publicGater),
	)
	if err != nil {
		return nil, err
	}

	// configure our own ping protocol
	pingService := &PingService{Host: host}
	host.SetStreamHandler(PingID, pingService.PingHandler)
	pubsubblocklist := pubsub.NewMapBlacklist()
	options := []pubsub.Option{pubsub.WithPeerExchange(false), pubsub.WithBlacklist(pubsubblocklist)}

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

	peerStatus := NewPeerStatus()
	rexnotification := make(chan RexNotification, 1)
	rexservice := NewRexService(host, peerStatus, nodeNetwork, ProtocolPrefix, rexnotification)
	rexservice.SetDelegate()

	nodeName := "default"
	psconnmgr := pubsubconn.InitPubSubConnMgr(ctx, ps, nodeName)

	newNode := &Node{NetworkName: nodeNetwork, Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery, Info: info, PubSubConnMgr: psconnmgr, peerStatus: peerStatus}

	go newNode.eventhandler(ctx)

	return newNode, nil
}

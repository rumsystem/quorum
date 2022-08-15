//go:build !js
// +build !js

package p2p

import (
	"context"
	"fmt"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/pkg/autorelay/audit"
)

type RelayNode struct {
	PeerID peer.ID
	Host   host.Host
	Info   *NodeInfo
}

func (node *RelayNode) GetRelay() *relayv2.Relay {
	bhost := node.Host.(*routedhost.RoutedHost).Unwrap().(*basichost.BasicHost)
	relayManager := bhost.RelayManager()
	return relayManager.Relay()
}

func (node *RelayNode) eventhandler(ctx context.Context) {
	evbus := node.Host.EventBus()
	subReachability, err := evbus.Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		networklog.Errorf("event subscribe err: %s:", err)
	}
	defer subReachability.Close()
	for {
		select {
		case ev := <-subReachability.Out():
			evt, ok := ev.(event.EvtLocalReachabilityChanged)
			if !ok {
				return
			}
			networklog.Infof("Reachability change: %s:", evt.Reachability.String())
			node.Info.NATType = evt.Reachability
		case <-ctx.Done():
			return
		}
	}
}

func NewRelayServiceNode(ctx context.Context, nodeOpt *options.RelayNodeOptions, key *ethkeystore.Key, listenAddresses []maddr.Multiaddr, db storage.QuorumStorage) (*RelayNode, error) {
	routingProtocol := fmt.Sprintf("%s/%s", ProtocolPrefix, nodeOpt.NetworkName)
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeServer),
			dht.Concurrency(10),
			dht.ProtocolPrefix(protocol.ID(routingProtocol)),
		)
		return dual.New(ctx, host, dhtOpts)
	})

	//privKey p2pcrypto.PrivKey
	ethprivkey := key.PrivateKey
	privkeybytes := ethcrypto.FromECDSA(ethprivkey)
	priv, err := p2pcrypto.UnmarshalSecp256k1PrivateKey(privkeybytes)
	if err != nil {
		return nil, err
	}

	identity := libp2p.Identity(priv)

	pstore, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, err
	}

	libp2poptions := []libp2p.Option{
		routing,
		libp2p.ListenAddrs(listenAddresses...),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.Ping(false),
		libp2p.Peerstore(pstore),
		libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(ws.New),
		),
		libp2p.DisableRelay(),
		libp2p.EnableRelayService(
			relay.WithAudit(audit.NewQuorumTrafficAudit(db)),
			relay.WithACL(NewQuorumRelayFilter(db)),
			relay.WithResources(nodeOpt.RC),
			relay.WithLimit(nil), /* double check, nodeOpt.RC.Limit should already be nil */
		),
		identity,
	}

	host, err := libp2p.New(
		libp2poptions...,
	)
	if err != nil {
		return nil, err
	}

	// configure our own ping protocol
	pingService := &PingService{Host: host}
	host.SetStreamHandler(PingID, pingService.PingHandler)

	info := &NodeInfo{NATType: network.ReachabilityUnknown}

	node := &RelayNode{Host: host, Info: info}

	go node.eventhandler(ctx)
	return node, nil
}

func (node *RelayNode) Bootstrap(ctx context.Context, bootstrapPeers cli.AddrList) error {
	return bootstrap(ctx, node.Host, bootstrapPeers)
}

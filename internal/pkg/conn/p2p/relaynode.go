//go:build !js
// +build !js

package p2p

import (
	"context"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	maddr "github.com/multiformats/go-multiaddr"
)

type RelayNode struct {
	PeerID peer.ID
	Host   host.Host
	Info   *NodeInfo
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

func NewRelayServiceNode(ctx context.Context, key *ethkeystore.Key, listenAddresses []maddr.Multiaddr) (*RelayNode, error) {
	var err error

	//privKey p2pcrypto.PrivKey
	ethprivkey := key.PrivateKey
	privkeybytes := ethcrypto.FromECDSA(ethprivkey)
	priv, err := p2pcrypto.UnmarshalSecp256k1PrivateKey(privkeybytes)
	if err != nil {
		return nil, err
	}

	identity := libp2p.Identity(priv)

	libp2poptions := []libp2p.Option{
		libp2p.ListenAddrs(listenAddresses...),
		libp2p.NATPortMap(),
		libp2p.Ping(false),
		libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(ws.New),
		),
		identity,
	}

	libp2poptions = append(libp2poptions,
		libp2p.DisableRelay(),
		libp2p.EnableRelayService(relay.WithLimit(nil)),
	)

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

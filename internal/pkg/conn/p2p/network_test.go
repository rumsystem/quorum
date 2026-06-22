//go:build !js
// +build !js

package p2p

import (
	"context"
	"testing"
	"time"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

func TestNodeConnecting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bootstrap := newTestNetworkNode(t, ctx, "bootstrap", true)
	peer1 := newTestNetworkNode(t, ctx, "peer1", false)
	peer2 := newTestNetworkNode(t, ctx, "peer2", false)
	defer closeTestNetworkNodes(bootstrap, peer1, peer2)

	bootstrapAddrs, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    bootstrap.Host.ID(),
		Addrs: bootstrap.Host.Addrs(),
	})
	if err != nil {
		t.Fatalf("build bootstrap peer address: %s", err)
	}

	for _, node := range []*Node{peer1, peer2} {
		if err := node.Bootstrap(ctx, cli.AddrList(bootstrapAddrs)); err != nil {
			t.Fatalf("bootstrap %s: %s", node.NodeName, err)
		}
		waitForConnectedPeer(t, ctx, node, bootstrap.Host.ID())
	}

	connected := peer1.AddPeers(ctx, []peer.AddrInfo{{
		ID:    peer2.Host.ID(),
		Addrs: peer2.Host.Addrs(),
	}})
	if connected != 1 {
		t.Fatalf("expected peer1 to connect to peer2 once, connected %d peers", connected)
	}
	waitForConnectedPeer(t, ctx, peer1, peer2.Host.ID())
	waitForConnectedPeer(t, ctx, peer2, peer1.Host.ID())
}

func newTestNetworkNode(t *testing.T, ctx context.Context, name string, isBootstrap bool) *Node {
	t.Helper()

	listenAddr, err := maddr.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	if err != nil {
		t.Fatalf("create listen address: %s", err)
	}
	privateKey, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate node key: %s", err)
	}
	cm, err := connmgr.NewConnManager(10, 100, connmgr.WithGracePeriod(time.Second))
	if err != nil {
		t.Fatalf("create connection manager: %s", err)
	}
	node, err := NewNode(
		ctx,
		name,
		&options.NodeOptions{NetworkName: "staten-test", MaxPeers: 10, ConnsHi: 100},
		isBootstrap,
		&ethkeystore.Key{PrivateKey: privateKey},
		cm,
		[]maddr.Multiaddr{listenAddr},
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("create %s node: %s", name, err)
	}
	if node.RoutingDiscovery == nil {
		t.Fatalf("%s node has no routing discovery", name)
	}
	return node
}

func waitForConnectedPeer(t *testing.T, ctx context.Context, node *Node, want peer.ID) {
	t.Helper()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		if node.Host.Network().Connectedness(want) == network.Connected {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("%s did not connect to %s: %s", node.NodeName, want, ctx.Err())
		case <-ticker.C:
		}
	}
}

func closeTestNetworkNodes(nodes ...*Node) {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Ddht != nil {
			_ = node.Ddht.Close()
		}
		if node.Host != nil {
			_ = node.Host.Close()
		}
	}
}

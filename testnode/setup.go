package testnode

import (
	"context"
	"fmt"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	"github.com/huo-ju/quorum/internal/pkg/utils"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	discovery "github.com/libp2p/go-libp2p-discovery"
	"log"
)

func Run2nodes(ctx context.Context, mockRendezvousString string) (*p2p.Node, *p2p.Node, *p2p.Node, error) {
	mockbootstrapaddr := "/ip4/127.0.0.1/tcp/8520"
	mockbootstrapnodekeys, err := localcrypto.NewKeys()
	listenaddresses, _ := utils.StringsToAddrs([]string{mockbootstrapaddr})
	node, err := p2p.NewNode(ctx, mockbootstrapnodekeys.PrivKey, connmgr.NewConnManager(1000, 50000, 30), listenaddresses, "")
	if err != nil {
		return nil, nil, nil, err
	}
	mockbootstrapp2paddr := fmt.Sprintf("%s/p2p/%s", mockbootstrapaddr, node.Host.ID())
	log.Printf("bootstrap:%s", mockbootstrapp2paddr)

	bootstrapaddrs, _ := utils.StringsToAddrs([]string{mockbootstrapp2paddr})
	defaultnodeconfig := &cli.Config{RendezvousString: mockRendezvousString, BootstrapPeers: bootstrapaddrs}

	mockpeer1nodekeys, err := localcrypto.NewKeys()
	peer1listenaddresses, _ := utils.StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/8551"})
	node1, err := p2p.NewNode(ctx, mockpeer1nodekeys.PrivKey, connmgr.NewConnManager(10, 200, 60), peer1listenaddresses, "")
	if err != nil {
		return nil, nil, nil, err
	}
	_ = node1.Bootstrap(ctx, *defaultnodeconfig)
	log.Println("Announcing peer1...")
	discovery.Advertise(ctx, node1.RoutingDiscovery, defaultnodeconfig.RendezvousString)
	log.Println("Successfully announced peer1!")

	//TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
	mockpeer2nodekeys, err := localcrypto.NewKeys()
	peer2listenaddresses, _ := utils.StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/8552"})
	node2, err := p2p.NewNode(ctx, mockpeer2nodekeys.PrivKey, connmgr.NewConnManager(10, 200, 60), peer2listenaddresses, "")
	if err != nil {
		return nil, nil, nil, err
	}
	node2.Bootstrap(ctx, *defaultnodeconfig)
	log.Println("Announcing peer2...")
	discovery.Advertise(ctx, node2.RoutingDiscovery, defaultnodeconfig.RendezvousString)
	log.Println("Successfully announced peer2")
	return node, node1, node2, nil
}

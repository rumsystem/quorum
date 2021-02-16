package p2p

import (
	"context"
	//"fmt"
	"github.com/golang/glog"
	blockstore "github.com/huo-ju/go-ipfs-blockstore"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	bitswap "github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
	"github.com/libp2p/go-libp2p"
	//autonat "github.com/libp2p/go-libp2p-autonat"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	//network "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	maddr "github.com/multiformats/go-multiaddr"
	"sync"
	"time"
)

type Node struct {
	PeerID           peer.ID
	Host             host.Host
	Pubsub           *pubsub.PubSub
	Ddht             *dual.DHT
	RoutingDiscovery *discovery.RoutingDiscovery
	Exchange         *bitswap.Bitswap
}

func NewNode(ctx context.Context, privKey p2pcrypto.PrivKey, bstore blockstore.Blockstore, listenAddresses []maddr.Multiaddr, jsontracerfile string) (*Node, error) {
	var ddht *dual.DHT
	var routingDiscovery *discovery.RoutingDiscovery
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeServer),
			dht.Concurrency(10),
		)

		var err error
		ddht, err = dual.New(ctx, host, dhtOpts)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	identity := libp2p.Identity(privKey)
	host, err := libp2p.New(ctx,
		routing,
		libp2p.ListenAddrs(listenAddresses...),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		identity,
	)
	if err != nil {
		return nil, err
	}

	var ps *pubsub.PubSub
	if jsontracerfile != "" {
		tracer, err := pubsub.NewJSONTracer(jsontracerfile)
		if err != nil {
			return nil, err
		}
		ps, err = pubsub.NewGossipSub(ctx, host, pubsub.WithEventTracer(tracer))
	} else {
		ps, err = pubsub.NewGossipSub(ctx, host)
	}

	if err != nil {
		return nil, err
	}

	bsnetwork := bsnet.NewFromIpfsHost(host, routingDiscovery)
	exchange := bitswap.New(ctx, bsnetwork, bstore)

	//newAutoNat, err := autonat.New(ctx, host, autonat.WithReachability(network.ReachabilityPublic))
	//autonataddr, err := newAutoNat.PublicAddr()
	//atuonatstatus := newAutoNat.Status()
	//glog.Infof("autonat %s", newAutoNat)
	//glog.Infof("autoant pubaddr %s", autonataddr)
	//glog.Infof("autoant pubaddr %s", atuonatstatus)
	//glog.Errorf("autonat err %s", err)

	newnode := &Node{Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery, Exchange: exchange.(*bitswap.Bitswap)}
	return newnode, nil
}

func (node *Node) FindPeers(ctx context.Context, RendezvousString string) ([]peer.AddrInfo, error) {
	pctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	var peers []peer.AddrInfo
	ch, err := node.RoutingDiscovery.FindPeers(pctx, RendezvousString)
	if err != nil {
		cancel()
		return nil, err
	}
	for pi := range ch {
		peers = append(peers, pi)
	}
	return peers, nil
}

func (node *Node) Bootstrap(ctx context.Context, config cli.Config) error {
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := node.Host.Connect(ctx, *peerinfo); err != nil {
				glog.Warning(err)
			} else {
				glog.Infof("Connection established with bootstrap node %s:", *peerinfo)
			}
		}()
	}
	wg.Wait()
	return nil
}

func (node *Node) ConnectPeers(ctx context.Context, config cli.Config) (int, error) {
	connectedCount := 0
	peers, err := node.FindPeers(ctx, config.RendezvousString)
	glog.Infof("find peers with Rendezvous %s ", config.RendezvousString)
	if err != nil {
		return connectedCount, err
	}
	for _, peer := range peers {
		if peer.ID == node.Host.ID() {
			continue
		}
		pctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		err := node.Host.Connect(pctx, peer)
		if err != nil {
			glog.Warningf("connect peer failure: %s \n", peer)
			cancel()
			continue
		} else {
			connectedCount++
			glog.Infof("connect: %s \n", peer)
		}
	}
	return connectedCount, nil
}

func (node *Node) EnsureConnect(ctx context.Context, rendezvousString string, f func()) {
	for {
		peers, _ := node.FindPeers(ctx, rendezvousString)
		glog.Infof("Find peers count: %d \n", len(peers))
		if len(peers) > 1 { // //connect 2 nodes at least
			break
		}
		time.Sleep(time.Second * 5)
	}
	f()
}

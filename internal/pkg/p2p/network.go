package p2p

import (
	"context"
	//"fmt"
	"github.com/golang/glog"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	"github.com/libp2p/go-libp2p"
	//autonat "github.com/libp2p/go-libp2p-autonat"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
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
}

func NewNode(ctx context.Context, isBootstrap bool, privKey p2pcrypto.PrivKey, cmgr *connmgr.BasicConnMgr, listenAddresses []maddr.Multiaddr, jsontracerfile string) (*Node, error) {
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
		libp2p.ConnectionManager(cmgr),
		identity,
	)
	if err != nil {
		return nil, err
	}

	options := []pubsub.Option{pubsub.WithFloodPublish(true), pubsub.WithPeerExchange(true)}

	if isBootstrap == true {
		// turn off the mesh in bootstrapnode
		pubsub.GossipSubD = 0
		pubsub.GossipSubDscore = 0
		pubsub.GossipSubDlo = 0
		pubsub.GossipSubDhi = 0
		pubsub.GossipSubDout = 0
		pubsub.GossipSubDlazy = 1024
		pubsub.GossipSubGossipFactor = 0.5
	}

	var ps *pubsub.PubSub
	if jsontracerfile != "" {
		tracer, err := pubsub.NewJSONTracer(jsontracerfile)
		if err != nil {
			return nil, err
		}
		options = append(options, pubsub.WithEventTracer(tracer))
	}
	ps, err = pubsub.NewGossipSub(ctx, host, options...)

	if err != nil {
		return nil, err
	}

	//bsnetwork := bsnet.NewFromIpfsHost(host, routingDiscovery)
	//exchange := bitswap.New(ctx, bsnetwork, bstore)
	newnode := &Node{Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery}
	return newnode, nil
}

func (node *Node) FindPeers(ctx context.Context, RendezvousString string) ([]peer.AddrInfo, error) {
	pctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	var peers []peer.AddrInfo
	ch, err := node.RoutingDiscovery.FindPeers(pctx, RendezvousString)
	if err != nil {
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

func (node *Node) ConnectPeers(ctx context.Context, peerok chan struct{}, config cli.Config) error {

	notify := false
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			//TODO: check peers status and max connect peers
			connectedCount := 0
			peers, err := node.FindPeers(ctx, config.RendezvousString)
			//glog.Infof("find peers with Rendezvous %s count: %d", config.RendezvousString, len(peers))
			if err != nil {
				return err
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
			if connectedCount > 0 {
				if notify == false {
					peerok <- struct{}{}
					notify = true
				}
			} else {
				glog.Infof("waitting for peers...")
			}
		}
	}
	return nil
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

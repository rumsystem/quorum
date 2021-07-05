package p2p

import (
	"context"
	"github.com/golang/glog"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	"github.com/huo-ju/quorum/internal/pkg/options"
	dsbadger2 "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p"
	//autonat "github.com/libp2p/go-libp2p-autonat"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"

	//network "github.com/libp2p/go-libp2p-core/network"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	maddr "github.com/multiformats/go-multiaddr"
	//basic "github.com/libp2p/go-libp2p/p2p/host/basic"
)

type NodeInfo struct {
	NATType network.Reachability
}

type Node struct {
	PeerID           peer.ID
	Host             host.Host
	Pubsub           *pubsub.PubSub
	Ddht             *dual.DHT
	Info             *NodeInfo
	RoutingDiscovery *discovery.RoutingDiscovery
}

func NewNode(ctx context.Context, nodeopt *options.NodeOptions, isBootstrap bool, ds *dsbadger2.Datastore, privKey p2pcrypto.PrivKey, cmgr *connmgr.BasicConnMgr, listenAddresses []maddr.Multiaddr, jsontracerfile string) (*Node, error) {
	var ddht *dual.DHT
	var routingDiscovery *discovery.RoutingDiscovery
	var pstore peerstore.Peerstore
	var err error
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeServer),
			dht.Concurrency(10),
			dht.ProtocolPrefix("/quorum"),
		)

		var err error
		ddht, err = dual.New(ctx, host, dhtOpts)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	identity := libp2p.Identity(privKey)

	libp2poptions := []libp2p.Option{routing,
		libp2p.ListenAddrs(listenAddresses...),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(cmgr),
		identity,
	}

	if ds != nil {
		pstore, err = pstoreds.NewPeerstore(ctx, ds, pstoreds.DefaultOpts())
		if err != nil {
			return nil, err
		}
		libp2poptions = append(libp2poptions, libp2p.Peerstore(pstore))
	}

	if nodeopt.EnableNat == true {
		libp2poptions = append(libp2poptions, libp2p.EnableNATService())
		glog.Infof("NAT enabled")
	}

	host, err := libp2p.New(ctx,
		libp2poptions...,
	)
	if err != nil {
		return nil, err
	}

	options := []pubsub.Option{pubsub.WithPeerExchange(true)}

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

	customprotocol := protocol.ID("/quorum/meshsub/1.1.0")
	protos := []protocol.ID{customprotocol}
	features := func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		if proto == customprotocol {
			return true
		}
		return false
	}

	options = append(options, pubsub.WithGossipSubProtocols(protos, features))

	ps, err = pubsub.NewGossipSub(ctx, host, options...)

	if err != nil {
		return nil, err
	}

	info := &NodeInfo{NATType: network.ReachabilityUnknown}

	newnode := &Node{Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery, Info: info}

	//reconnect peers

	storedpeers := []peer.AddrInfo{}
	if ds != nil {
		for _, peer := range pstore.Peers() {
			peerinfo := pstore.PeerInfo(peer)
			storedpeers = append(storedpeers, peerinfo)
		}
	}
	if len(storedpeers) > 0 {
		//TODO: try connect every x minutes for x*y minutes?
		go func() {
			newnode.AddPeers(ctx, storedpeers)
		}()
	}
	go newnode.eventhandler(ctx)

	return newnode, nil
}

func (node *Node) eventhandler(ctx context.Context) {
	evbus := node.Host.EventBus()
	subReachability, err := evbus.Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		glog.Errorf("event subscribe err: %s:", err)
	}
	defer subReachability.Close()
	for {
		select {
		case ev := <-subReachability.Out():
			evt, ok := ev.(event.EvtLocalReachabilityChanged)
			if !ok {
				return
			}
			glog.Infof("Reachability change: %s:", evt.Reachability.String())
			node.Info.NATType = evt.Reachability
		case <-ctx.Done():
			return
		}
	}
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

func (node *Node) AddPeers(ctx context.Context, peers []peer.AddrInfo) int {
	connectedCount := 0
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
			glog.Infof("connect: %s", peer)
		}
	}
	return connectedCount
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
					//glog.Infof("connect: %s \n", peer)
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

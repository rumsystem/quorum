package p2p

import (
    "time"
    "context"
	"sync"
	"github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-discovery"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/golang/glog"
    "github.com/huo-ju/quorum/internal/pkg/cli"
)

type Node struct {
    Ctx context.Context
    PeerID peer.ID
    Host host.Host
    Pubsub *pubsub.PubSub
	Ddht *dual.DHT
	RoutingDiscovery *discovery.RoutingDiscovery
}


func NewNode(ctx context.Context, privKey p2pcrypto.PrivKey, listenAddresses []maddr.Multiaddr, jsontracerfile string) (*Node, error){
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
    }else {
	    ps, err = pubsub.NewGossipSub(ctx, host)
    }

    if err != nil {
        return nil, err
    }
    newnode := &Node{Ctx: ctx, Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery}
    return newnode,nil
}

func (node *Node) FindPeers(RendezvousString string)  ([]peer.AddrInfo, error) {
    pctx, _ := context.WithTimeout(node.Ctx, time.Second*10)
    var peers []peer.AddrInfo
    ch, err := node.RoutingDiscovery.FindPeers(pctx, RendezvousString)
	if err != nil {
        return nil, err
	}
	for pi := range ch {
	    peers= append(peers, pi)
	}
    return peers, nil
}


func (node *Node) Bootstrap(config cli.Config)  (error) {
    var wg sync.WaitGroup
    for _, peerAddr := range config.BootstrapPeers {
        peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
        wg.Add(1)
        go func() {
	    defer wg.Done()
	        if err := node.Host.Connect(node.Ctx, *peerinfo); err != nil {
                glog.Warning(err)
	        } else {
                glog.Infof("Connection established with bootstrap node %s:", *peerinfo)
	        }
        }()
    }
    wg.Wait()
    return nil
}

func (node *Node) ConnectPeers(config cli.Config) (int, error){
    connectedCount := 0
    peers, err := node.FindPeers(config.RendezvousString)
	glog.Infof("find peers with Rendezvous %s ", config.RendezvousString)
    if err != nil {
        return connectedCount, err
    }
    for _, peer := range peers {
	    if peer.ID == node.Host.ID() {
	        continue
	    }
	    //pctx, _ := context.WithTimeout(ctx, time.Second*10)
        err := node.Host.Connect(node.Ctx, peer)
        if err != nil {
            glog.Warningf("connect peer failure: %s \n", peer)
        }else {
	        connectedCount++
            glog.Infof("connect: %s \n", peer)
        }
    }
    return connectedCount, nil
}

func (node *Node) EnsureConnect(rendezvousString string, f func()) {
	for {
		peers, _:= node.FindPeers(rendezvousString)
        glog.Infof("Find peers count: %d \n", len(peers))
	    if len(peers)>1 { // //connect 2 nodes at least
	        break
	    }
	    time.Sleep(time.Second * 5)
	}
    f()
}

package p2p

import (
    "time"
    "context"
	"github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-discovery"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	maddr "github.com/multiformats/go-multiaddr"
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
        var err error
        ddht, err = dual.New(ctx, host)
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

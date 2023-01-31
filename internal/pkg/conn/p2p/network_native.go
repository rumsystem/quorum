//go:build !js
// +build !js

package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	discoveryrouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

var peerChan = make(chan peer.AddrInfo)

func GetRelayPeerChan() chan peer.AddrInfo {
	return peerChan
}

func NewNode(ctx context.Context, nodename string, nodeopt *options.NodeOptions, isBootstrap bool, key *ethkeystore.Key, cmgr *connmgr.BasicConnMgr, listenAddresses []maddr.Multiaddr, skippeers []string, jsontracerfile string) (*Node, error) {
	var ddht *dual.DHT
	var routingDiscovery *discoveryrouting.RoutingDiscovery
	var err error

	ethprivkey := key.PrivateKey
	privkeybytes := ethcrypto.FromECDSA(ethprivkey)
	priv, err := p2pcrypto.UnmarshalSecp256k1PrivateKey(privkeybytes)
	if err != nil {
		return nil, err
	}

	nodenetworkname := nodeopt.NetworkName
	if nodeopt.EnableDevNetwork == true {
		nodenetworkname = fmt.Sprintf("%s-%s", nodeopt.NetworkName, "dev")
	}

	routingcustomprotocol := fmt.Sprintf("%s/%s", ProtocolPrefix, nodenetworkname)
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeServer),
			dht.Concurrency(10),
			dht.ProtocolPrefix(protocol.ID(routingcustomprotocol)),
		)

		var err error
		ddht, err = dual.New(ctx, host, dhtOpts)
		routingDiscovery = discoveryrouting.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	networklog.Infof("Enable dht protocol prefix: %s", routingcustomprotocol)

	identity := libp2p.Identity(priv)

	libp2poptions := []libp2p.Option{routing,
		libp2p.ListenAddrs(listenAddresses...),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(cmgr),
		libp2p.Ping(false),
		libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(ws.New),
		),
		identity,
	}

	if nodeopt.EnableRelay {
		libp2poptions = append(libp2poptions,
			libp2p.EnableAutoRelay(
				autorelay.WithPeerSource(func(context.Context, int) <-chan peer.AddrInfo { return peerChan }, time.Hour),
				autorelay.WithMaxCandidates(1),
				autorelay.WithNumRelays(99999),
				autorelay.WithBootDelay(0)),
		)
	}

	pstore, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, err
	}
	libp2poptions = append(libp2poptions, libp2p.Peerstore(pstore))

	if nodeopt.EnableNat == true {
		libp2poptions = append(libp2poptions, libp2p.EnableNATService())
		networklog.Infof("NAT enabled")
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

	pubsubblocklist := pubsub.NewMapBlacklist()

	for _, sp := range skippeers {
		spid, err := peer.Decode(sp)
		if err != nil {
			fmt.Println("===decode peerid err:", err)
		}
		fmt.Println("===add black list:", spid)
		pubsubblocklist.Add(spid)
	}
	options := []pubsub.Option{pubsub.WithPeerExchange(true), pubsub.WithPeerOutboundQueueSize(128), pubsub.WithBlacklist(pubsubblocklist)}

	networklog.Infof("Network Name %s", nodenetworkname)
	if isBootstrap {
		// turn off the mesh in bootstrapnode and relay node
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

	customprotocol := protocol.ID(fmt.Sprintf("%s/meshsub/1.1.0", fmt.Sprintf("%s/%s", ProtocolPrefix, nodenetworkname)))
	protos := []protocol.ID{customprotocol}
	features := func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		if proto == customprotocol {
			return true
		}
		return false
	}

	networklog.Infof("Enable protocol: %s", customprotocol)

	options = append(options, pubsub.WithGossipSubProtocols(protos, features))
	options = append(options, pubsub.WithPeerOutboundQueueSize(128))

	ps, err = pubsub.NewGossipSub(ctx, host, options...)

	if err != nil {
		return nil, err
	}

	//commented by cuicat
	// enable pubsub ping
	//psPing := NewPSPingService(ctx, ps, host.ID())
	//psPing.EnablePing()

	info := &NodeInfo{NATType: network.ReachabilityUnknown}
	newnode := &Node{NetworkName: nodenetworkname, NodeName: nodename, Host: host, SkipPeers: skippeers, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery, Info: info, Nodeopt: nodeopt}

	go newnode.eventhandler(ctx)
	return newnode, nil
}

func (node *Node) Bootstrap(ctx context.Context, bootstrapPeers cli.AddrList) error {
	return bootstrap(ctx, node.Host, bootstrapPeers)
}

func bootstrap(ctx context.Context, h host.Host, addrs cli.AddrList) error {
	var wg sync.WaitGroup
	for _, peerAddr := range addrs {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				networklog.Warning(err)
			} else {
				networklog.Infof("Connection established with bootstrap node %s:", *peerinfo)
			}
		}()
	}
	wg.Wait()
	return nil
}

func (node *Node) ConnectPeers(ctx context.Context, peerok chan struct{}, maxpeers int, rendezvousStr string) error {
	notify := false
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			//TODO: check peers status and max connect peers
			connectedCount := 0
			if notify == false {
				peers, err := node.FindPeers(ctx, rendezvousStr)
				if err != nil {
					return err
				}
				for _, peer := range peers {
					if peer.ID == node.Host.ID() {
						continue
					}
					skip := false
					for _, sp := range node.SkipPeers {
						if sp == peer.ID.Pretty() {
							skip = true
						}
					}
					if skip == true {
						continue
					}
					pctx, cancel := context.WithTimeout(ctx, time.Second*10)
					defer cancel()
					err := node.Host.Connect(pctx, peer)
					if err != nil {
						networklog.Warningf("connect peer failure: %s", peer)
						cancel()
						continue
					} else {
						connectedCount++
					}
				}
			}
			if connectedCount >= maxpeers {
				if notify == false {
					peerok <- struct{}{}
					notify = true
				}
			} else {
				networklog.Infof("finding peers...")
			}
		}
	}
	return nil
}

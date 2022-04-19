//go:build !js
// +build !js

package p2p

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	dsbadger2 "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/pubsubconn"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

func NewNode(ctx context.Context, nodename string, nodeopt *options.NodeOptions, isBootstrap bool, ds *dsbadger2.Datastore, key *ethkeystore.Key, cmgr *connmgr.BasicConnMgr, listenAddresses []maddr.Multiaddr, jsontracerfile string) (*Node, error) {
	var ddht *dual.DHT
	var routingDiscovery *discovery.RoutingDiscovery
	var pstore peerstore.Peerstore
	var err error

	//privKey p2pcrypto.PrivKey
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
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
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

	if nodeopt.EnableRelay && !nodeopt.EnableRelayService {
		relayServerAddr := maddr.StringCast("/ip4/167.114.61.176/tcp/33334/p2p/16Uiu2HAmSJAg2hwhbAEHy63o1AjwXQ2ERkwZQp1EfoVxR3uZaBGh")

		relayServer, err := peer.AddrInfoFromP2pAddr(relayServerAddr)
		if err != nil {
			panic(err)
		}
		staticRelays := []peer.AddrInfo{*relayServer}
		libp2poptions = append(libp2poptions,
			libp2p.EnableAutoRelay(
				autorelay.WithStaticRelays(staticRelays),
				autorelay.WithMaxCandidates(1),
				autorelay.WithNumRelays(99999),
				autorelay.WithBootDelay(0)),
		)
	}
	if nodeopt.EnableRelayService {
		libp2poptions = append(libp2poptions,
			libp2p.DisableRelay(),
			libp2p.EnableRelayService(),
			libp2p.ForceReachabilityPublic(),
			libp2p.AddrsFactory(func(addrs []maddr.Multiaddr) []maddr.Multiaddr {
				for i, addr := range addrs {
					saddr := addr.String()
					if strings.HasPrefix(saddr, "/ip4/127.0.0.1/") {
						addrNoIP := strings.TrimPrefix(saddr, "/ip4/127.0.0.1")
						addrs[i] = maddr.StringCast("/dns4/localhost" + addrNoIP)
					}
				}
				return addrs
			}),
		)
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
	options := []pubsub.Option{pubsub.WithPeerExchange(true), pubsub.WithPeerOutboundQueueSize(128), pubsub.WithBlacklist(pubsubblocklist)}

	networklog.Infof("Network Name %s", nodenetworkname)
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

	psping := NewPSPingService(ctx, ps, host.ID())
	psping.EnablePing()
	info := &NodeInfo{NATType: network.ReachabilityUnknown}

	psconnmgr := pubsubconn.InitPubSubConnMgr(ctx, ps, nodename)

	if isBootstrap == false && nodeopt.EnableRumExchange == true {
	}

	newnode := &Node{NetworkName: nodenetworkname, Host: host, Pubsub: ps, Ddht: ddht, RoutingDiscovery: routingDiscovery, Info: info, PubSubConnMgr: psconnmgr}
	//RumExchange: rexservice

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

func (node *Node) SetRumExchange(ctx context.Context, dbmgr *storage.DbMgr) {
	peerStatus := NewPeerStatus()
	var rexnotification chan RexNotification
	rexnotification = make(chan RexNotification, 1)
	var rexservice *RexService
	rexservice = NewRexService(node.Host, peerStatus, node.NetworkName, ProtocolPrefix, rexnotification)
	rexservice.SetDelegate()
	rexchaindata := NewRexChainData(rexservice)
	rexrelay := NewRexRelay(rexservice, dbmgr)
	rexservice.SetHandlerMatchMsgType("rumchaindata", rexchaindata.Handler)
	rexservice.SetHandlerMatchMsgType("rumrelay", rexrelay.Handler)
	networklog.Infof("Enable protocol RumExchange")

	node.peerStatus = peerStatus
	node.RumExchange = rexservice

	if rexnotification != nil {
		go node.rexhandler(ctx, rexnotification)
	}
}

func (node *Node) Bootstrap(ctx context.Context, config cli.Config) error {
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := node.Host.Connect(ctx, *peerinfo); err != nil {
				networklog.Warning(err)
			} else {
				networklog.Infof("Connection established with bootstrap node %s:", *peerinfo)
			}
		}()
	}
	wg.Wait()
	return nil
}

func (node *Node) ConnectPeers(ctx context.Context, peerok chan struct{}, maxpeers int, config cli.Config) error {
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
				peers, err := node.FindPeers(ctx, config.RendezvousString)
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
						networklog.Warningf("connect peer failure: %s \n", peer)
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

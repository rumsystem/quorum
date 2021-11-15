package p2p

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const ProtocolPrefix string = "/quorum"

var networklog = logging.Logger("network")

type NodeInfo struct {
	NATType network.Reachability
}

type Node struct {
	PeerID           peer.ID
	Host             host.Host
	NetworkName      string
	Pubsub           *pubsub.PubSub
	Ddht             *dual.DHT
	Info             *NodeInfo
	RoutingDiscovery *discovery.RoutingDiscovery
}

func (node *Node) eventhandler(ctx context.Context) {
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
			networklog.Warningf("connect peer failure: %s \n", peer)
			cancel()
			continue
		} else {
			connectedCount++
			networklog.Infof("connect: %s", peer)
		}
	}
	return connectedCount
}

func (node *Node) EnsureConnect(ctx context.Context, rendezvousString string, f func()) {
	for {
		peers, _ := node.FindPeers(ctx, rendezvousString)
		networklog.Infof("Find peers count: %d \n", len(peers))
		if len(peers) > 1 { // //connect 2 nodes at least
			break
		}
		time.Sleep(time.Second * 5)
	}
	f()
}

func (node *Node) PeersProtocol() *map[string][]string {
	protocolpeers := make(map[string][]string)
	peerstore := node.Host.Peerstore()
	peers := peerstore.Peers()
	for _, peerid := range peers {
		if node.Host.Network().Connectedness(peerid) == network.Connected {
			if node.Host.ID() != peerid {
				conns := node.Host.Network().ConnsToPeer(peerid)
				for _, c := range conns {
				check:
					for _, s := range c.GetStreams() {
						if string(s.Protocol()) != "" {
							if protocolpeers[string(s.Protocol())] == nil {
								protocolpeers[string(s.Protocol())] = []string{peerid.String()}
							} else {
								for _, id := range protocolpeers[string(s.Protocol())] {
									if id == peerid.String() {
										break check
									}
								}
								protocolpeers[string(s.Protocol())] = append(protocolpeers[string(s.Protocol())], peerid.String())
							}
						}
					}
				}
			}
		}
	}
	return &protocolpeers
}

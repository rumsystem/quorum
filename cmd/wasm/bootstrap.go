package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"

	ws "github.com/libp2p/go-ws-transport"
	maddr "github.com/multiformats/go-multiaddr"
)

type addrList []maddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := maddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

func StringsToAddrs(addrStrings []string) (maddrs []maddr.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := maddr.NewMultiaddr(addrString)
		if err != nil {
			println(err.Error())
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bootAddrs, _ := StringsToAddrs([]string{"/ip4/0.0.0.0/tcp/4000/ws"})

	// WebSockets only:
	h, _ := libp2p.New(
		ctx,
		libp2p.ListenAddrs(bootAddrs...),
		libp2p.Transport(ws.New),
	)

	//protocolID := "/p2p/quorum"
	//h.SetStreamHandler(protocol.ID(protocolID), handleStream)

	kademliaDHT, _ := dht.New(ctx, h, dht.Option(dht.Mode(dht.ModeServer)))

	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, "test")
	println("Successfully announced!")

	for _, addr := range h.Addrs() {
		println(addr.String() + "/p2p/" + h.ID().String())
	}

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Now, look for others who have announced
				// This is like your friend telling you the location to meet you.
				peerChan, err := routingDiscovery.FindPeers(ctx, "test")
				if err != nil {
					panic(err)
				}

				for peer := range peerChan {
					if peer.ID == h.ID() {
						// println("Found peer(self):", peer.String())
					} else {
						println("Found peer:", peer.String())
					}
				}
			}
		}

	}()

	evbus := h.EventBus()
	subReachability, err := evbus.Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		fmt.Errorf("event subscribe err: %s:", err)
	}
	defer subReachability.Close()
	for {
		select {
		case ev := <-subReachability.Out():
			evt, ok := ev.(event.EvtLocalReachabilityChanged)
			if !ok {
				return
			}
			fmt.Printf("Reachability change: %s:", evt.Reachability.String())
		case <-ctx.Done():
			return
		}
	}

}

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"
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
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Bootstrap Node Url: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bootAddrs, _ := StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/4001/ws"})

	// WebSockets only:
	h, _ := libp2p.New(
		ctx,
		libp2p.ListenAddrs(bootAddrs...),
		libp2p.Transport(ws.New),
	)

	//protocolID := "/p2p/quorum"
	//h.SetStreamHandler(protocol.ID(protocolID), handleStream)

	kademliaDHT, _ := dht.New(ctx, h)

	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	as, _ := StringsToAddrs([]string{url})

	var wg sync.WaitGroup
	for _, peerAddr := range as {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			println("connect: ", peerAddr)
			if err := h.Connect(ctx, *peerinfo); err != nil {
				println(err.Error())
			} else {
				println("Connection established with bootstrap node: ", peerinfo.String())
			}
		}()
	}
	wg.Wait()

	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, "test")
	println("Successfully announced!")

	for _, addr := range h.Addrs() {
		println(addr.String() + "/" + h.ID().String())
	}

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Now, look for others who have announced
				// This is like your friend telling you the location to meet you.
				println("Searching for other peers...")
				peerChan, err := routingDiscovery.FindPeers(ctx, "test")
				if err != nil {
					panic(err)
				}

				for peer := range peerChan {
					if peer.ID == h.ID() {
						println("Found peer(self):", peer.String())
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

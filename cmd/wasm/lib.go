package main

import (
	"context"
	"strings"
	"syscall/js"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ws "github.com/libp2p/go-ws-transport"
	maddr "github.com/multiformats/go-multiaddr"
)

// quit channel
var qChan = make(chan struct{}, 0)

func registerCallbacks() {
	js.Global().Set("StartQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan == nil {
			qChan = make(chan struct{}, 0)
		}
		bootAddr := args[0].String()
		go StartQuorum(bootAddr)
		return js.ValueOf(true).Bool()
	}))

	js.Global().Set("StopQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan != nil {
			close(qChan)
			qChan = nil
		}
		return js.ValueOf(true).Bool()
	}))

	js.Global().Set("WSTest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan == nil {
			qChan = make(chan struct{}, 0)
		}
		WSTest()
		return js.ValueOf(true).Bool()
	}))
}

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

type Config struct {
	RendezvousString string
	BootstrapPeers   addrList
}

func GetConfig(boot string) Config {
	bootAddrs, _ := StringsToAddrs([]string{boot})
	var DefaultConfig = Config{
		"e6629921-b5cd-4855-9fcd-08bcc39caef7",
		bootAddrs,
	}
	return DefaultConfig
}

func WSTest() {
	go func() {
		openSignal := make(chan struct{})
		ws := js.Global().Get("WebSocket").New("ws://127.0.0.1:4000")
		ws.Set("binaryType", "arraybuffer")

		ws.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			println("opened!!")
			close(openSignal)
			return nil
		}))

		messageHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			arrayBuffer := args[0].Get("data")
			data := arrayBufferToBytes(arrayBuffer)
			println(data)
			return nil
		})
		ws.Call("addEventListener", "message", messageHandler)

		// this will block, and websocket will never open
		// do not do this
		<-openSignal
		println("openSignal fired")

	}()
}

func arrayBufferToBytes(buffer js.Value) []byte {
	view := js.Global().Get("Uint8Array").New(buffer)
	dataLen := view.Length()
	data := make([]byte, dataLen)
	if js.CopyBytesToGo(data, view) != dataLen {
		panic("expected to copy all bytes")
	}
	return data
}

func StartQuorum(bootAddr string) {
	ctx, cancel := context.WithCancel(context.Background())
	config := GetConfig(bootAddr)

	var routingDiscovery *discovery.RoutingDiscovery
	routeProtoPrefix := "/quorum/nevis"
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeClient),
			dht.Concurrency(10),
			dht.ProtocolPrefix(protocol.ID(routeProtoPrefix)),
		)

		var err error
		ddht, err := dual.New(ctx, host, dhtOpts)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	// WebSockets only:
	h, err := libp2p.New(
		ctx,
		routing,
		libp2p.Transport(ws.New),
		libp2p.ListenAddrs(),
	)
	if err != nil {
		panic(err)
	}

	println("id: ", h.ID().String())

	psOptions := []pubsub.Option{pubsub.WithPeerExchange(true)}

	qProto := protocol.ID("/quorum/nevis/meshsub/1.1.0")
	protos := []protocol.ID{qProto}
	features := func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		if proto == qProto {
			return true
		}
		return false
	}
	psOptions = append(psOptions, pubsub.WithGossipSubProtocols(protos, features))

	psOptions = append(psOptions, pubsub.WithPeerOutboundQueueSize(128))

	ps, err := pubsub.NewGossipSub(ctx, h, psOptions...)

	println(ps)

	//kademliaDHT, _ := dht.New(ctx, h, dht.Option(dht.Mode(dht.ModeClient)))

	//if err := kademliaDHT.Bootstrap(ctx); err != nil {
	//	panic(err)
	//}

	// Do not use sync.WaitGroup, it will block this thread
	// and the socket will never opened
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		url := peerAddr.String()
		println("connecting: ", url)
		if err := h.Connect(ctx, *peerinfo); err != nil {
			panic(err)
		} else {
			println("Connection established with bootstrap node: ", url)

		}
	}

	println("Announcing ourselves...")
	discovery.Advertise(ctx, routingDiscovery, config.RendezvousString)
	println("Successfully announced!")

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Now, look for others who have announced
				// This is like your friend telling you the location to meet you.
				println("Searching for other peers...")
				peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
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
			case <-qChan:
				cancel()
			}
		}
	}()
}

func main() {
	c := make(chan struct{}, 0)

	println("WASM Go Initialized")
	// register functions
	registerCallbacks()
	<-c
}

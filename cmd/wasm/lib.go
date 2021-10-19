package main

import (
	"context"
	"strings"
	"syscall/js"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
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
		StartQuorum(bootAddr)
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

func handleStream(stream network.Stream) {
	println("Got a new stream!")
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
	ProtocolID       string
}

func GetConfig(boot string) Config {
	bootAddrs, _ := StringsToAddrs([]string{boot})
	var DefaultConfig = Config{
		"test",
		bootAddrs,
		"/quorum/test",
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
	// defer cancel()

	// config := GetConfig("/ip4/127.0.0.1/tcp/3001/ws", "/ip4/127.0.0.1/tcp/3002/ws")
	config := GetConfig(bootAddr)

	// id, _ := peer.IDFromPrivateKey(priv)
	//priv, _, _ := test.RandTestKeyPair(crypto.Ed25519, 256)
	// id, _ := peer.IDFromPrivateKey(priv)

	// WebSockets only:
	h, err := libp2p.New(
		ctx,
		libp2p.Transport(ws.New),
		libp2p.ListenAddrs(),
	)
	if err != nil {
		panic(err)
	}

	println("id: ", h.ID().String())

	protocolID := "/p2p/quorum"
	h.SetStreamHandler(protocol.ID(protocolID), handleStream)

	kademliaDHT, _ := dht.New(ctx, h, dht.Option(dht.Mode(dht.ModeClient)))

	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		url := peerAddr.String()
		// wg.Add(1)
		go func() {
			// defer wg.Done()
			println("connecting: ", url)
			if err := h.Connect(ctx, *peerinfo); err != nil {
				panic(err)
			} else {
				println("Connection established with bootstrap node: ", url)
			}
		}()
	}
	// wg.Wait()

	println("Announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
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

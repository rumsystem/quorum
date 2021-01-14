package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
    "time"
    "strings"
	"os"
	"io"
	"sync"
	"crypto/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-kad-dht/dual"

	"github.com/libp2p/go-libp2p-core/crypto"
	//dht "github.com/libp2p/go-libp2p-kad-dht"
	multiaddr "github.com/multiformats/go-multiaddr"
	//logging "github.com/whyrusleeping/go-logging"

	"github.com/ipfs/go-log"
)

var logger = log.Logger("rendezvous")
var inputData *InputData

// State 0 not ready 1 text message 2 bin data stream
type InputData struct{
   State int
}


func handleStream(stream network.Stream) {
	logger.Info("Got a new stream!")
    inputData.State = 1 //set to message state

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
    mode := 1 //set to command mode
    var filewriter *os.File
    //var filewriter *bufio.Writer
	for {
		buf := make([]byte, 500)
		n, err := rw.Read(buf)
        if buf[0] == '*' && mode ==1 {
            fmt.Println("receive a command: %s\n", string(buf))
        } else {
            fmt.Println("is data stream.")
            if mode == 1 { //switch from command mode, create new file to receve data stream
                path := "incomefiles/income.bin"
                filewriter, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
                //file, err := os.Create(path)
                if err != nil {
                    fmt.Println("create file error")
                    fmt.Println(err)
                }
            }
            mode = 2 //set to data mode
		    fmt.Printf("income length: %d\n", n)
		    if err != nil {
			    if err == io.EOF {
			        fmt.Printf("=======EOF :%d \n", n)
                    filewriter.Write(buf[:n])
                    mode = 1 //set to command mode
			        return
			    }
		        fmt.Println("Error reading from buffer")
		        panic(err)
		    } else {
                if filewriter != nil {
                    if n == len(buf){
                        fmt.Printf("write length %d\n", n)
                        filewriter.Write(buf)
                    }else {
                        fmt.Printf("write length %d\n", n)
                        filewriter.Write(buf[:n])
                    }
                }
            }
        }

		if n == 0 {
            fmt.Println("return")
		    return
		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}
        fmt.Printf("stdin data: %s \n", sendData)
        if sendData[0]=='*' {
            fmt.Println("command input: %s",sendData)
            if strings.Index(sendData, "*file") ==0 {
                filename := strings.TrimSpace(string(sendData[5:]))
                fmt.Printf("send file: %s \n", filename)
				f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
				fmt.Println(f)
				if err != nil {
					logger.Fatal(err)
				}
				filereader := bufio.NewReader(f)
				io.Copy(rw, filereader)
				err = rw.Flush()
				fmt.Printf("write data stream...")
				fmt.Println(err)

				if err := f.Close(); err != nil {
					logger.Fatal(err)
				}

            }else {
		        _, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		        if err != nil {
		            fmt.Println("Error writing to buffer")
		            panic(err)
		        }
		        err = rw.Flush()
		        if err != nil {
		            fmt.Println("Error flushing buffer")
		            panic(err)
		        }
            }
        } else {
		    fmt.Println("undefined command")
        }


	}
}

func main() {
	//log.SetAllLoggers(logging.WARNING)
	log.SetLogLevel("rendezvous", "info")
	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()

    inputData = &InputData{}
	if err != nil {
		panic(err)
	}

	if *help {
		fmt.Println("This program demonstrates a simple p2p chat application using libp2p")
		fmt.Println()
		fmt.Println("Usage: Run './chat in two different terminals. Let them connect to the bootstrap nodes, announce themselves and connect to the peers")
		flag.PrintDefaults()
		return
	}

	ctx := context.Background()

	var ddht *dual.DHT
	var routingDiscovery *discovery.RoutingDiscovery

	//routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
	//	return dual.New(ctx, host)
	//})


	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		panic(err)
	}
	identity := libp2p.Identity(priv)
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		var err error
		ddht, err = dual.New(ctx, host)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)

		return ddht, err
	})

	// libp2p.New constructs a new libp2p Host. Other options can be added
	// here.
	host, err := libp2p.New(ctx,
		routing,
		libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...),
		identity,
	)
	if err != nil {
		panic(err)
	}
	logger.Info("Host created. We are:", host.ID())
	logger.Info(host.Addrs())

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	//kademliaDHT, err := dht.New(ctx, host)
	//if err != nil {
	//	panic(err)
	//}

    //fmt.Printf("%s", []multiaddr.Multiaddr(config.ListenAddresses))
    for _, addr := range config.ListenAddresses {
        fmt.Printf("Bootstrap ID: %s/p2p/%s\n" ,addr , host.ID().Pretty())
    }
	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	logger.Debug("Bootstrapping the DHT")
	//if err = kademliaDHT.Bootstrap(ctx); err != nil {
	//	panic(err)
	//}

	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				logger.Warning(err)
			} else {
				logger.Info("Connection established with bootstrap node:", *peerinfo)
			}
		}()
	}
	wg.Wait()


	// We use a rendezvous point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	logger.Info("Announcing ourselves...")
	discovery.Advertise(ctx, routingDiscovery, config.RendezvousString)
	//routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	//discovery.Advertise(ctx, routingDiscovery, config.RendezvousString)
	logger.Info("Successfully announced!")

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	logger.Info("Searching for other peers...")
    fmt.Println(config.RendezvousString)
	fmt.Println("DHT in a bootstrapped state")
	time.Sleep(time.Second * 5)

    fmt.Println("Lan Routing Table:")
	ddht.LAN.RoutingTable().Print()
    fmt.Println("Wan Routing Table:")
	ddht.WAN.RoutingTable().Print()

	pctx, _ := context.WithTimeout(ctx, time.Second*10)
	peers, err := discovery.FindPeers(pctx, routingDiscovery, config.RendezvousString)
	if err != nil {
		panic(err)
	}

	for _, peer := range peers {
		if peer.ID == host.ID() {
			continue
		}
		logger.Debug("Found peer:", peer)

		logger.Debug("Connecting to:", peer)
		stream, err := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

		if err != nil {
			logger.Warning("Connection failed:", err)
			continue
		} else {
            inputData.State = 1 //set to message state
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			go writeData(rw)
			go readData(rw)
		}

		logger.Info("Connected to:", peer)
	}

	select {}
}

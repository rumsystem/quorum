package main

import (
	"fmt"
	"os"
	"bufio"
    "io"
	"syscall"
	"flag"
	"context"
    "crypto/rand"
	"os/signal"
	"path/filepath"
	"github.com/spf13/viper"
	"github.com/golang/glog"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	//peerstore "github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var (
	rootDir string
)

func loadconf() {
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.AddConfigPath(filepath.Dir("."))
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.ReadInConfig()
	rootDir = viper.GetString("ROOT_DIR")
}

func doEcho(s network.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}
	glog.Infof("read: %s\n", str)
	_, err = s.Write([]byte("echo "))
	_, err = s.Write([]byte(str))
	return err
}

func main() {
	flag.Parse()
	glog.V(2).Infof("Start...")
	loadconf()
    fmt.Println(rootDir)
	// create a background context (i.e. one that never cancels)
	ctx := context.Background()

    var r io.Reader = rand.Reader
    priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
    fmt.Println(priv)

	// start a libp2p node with default settings
    //node, err := libp2p.New(ctx,
    //        libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	//		libp2p.Ping(false),
    //)

    listenPort := 2000
    opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
    node, err := libp2p.New(ctx, opts...)

	if err != nil {
		panic(err)
	}

    hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", node.ID().Pretty()))
    addr := node.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)

    fmt.Println(fullAddr)


	// configure our own ping protocol
	pingService := &ping.PingService{Host: node}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)

	node.SetStreamHandler("/echo/1.0.0", func(s network.Stream) {
		glog.Infof("Got a new stream!")
		if err := doEcho(s); err != nil {
			glog.Error(err)
			s.Reset()
		} else {
			s.Close()
		}
	})



	// print the node's listening addresses
	fmt.Println("Listen addresses:", node.Addrs())

	//peerInfo := peerstore.AddrInfo{
	//	ID:    node.ID(),
	//	Addrs: node.Addrs(),
	//}
	//addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	//fmt.Println("libp2p node address:", addrs[0])

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")

	// shut the node down
	if err := node.Close(); err != nil {
		panic(err)
	}

}


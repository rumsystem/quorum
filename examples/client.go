package main

import (
	"fmt"
	"os"
	"flag"
	"context"
	"io/ioutil"
	"path/filepath"
	"github.com/spf13/viper"
	"github.com/golang/glog"
    "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
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

func main() {
	flag.Parse()
	glog.V(2).Infof("Start...")
	loadconf()
	// create a background context (i.e. one that never cancels)
	ctx := context.Background()

	// start a libp2p node with default settings
    node, err := libp2p.New(ctx,
            libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
			libp2p.Ping(false),
    )
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 {
		ipfsaddr, err := multiaddr.NewMultiaddr(os.Args[1])
		if err != nil {
			panic(err)
		}

		pid, err := ipfsaddr.ValueForProtocol(multiaddr.P_IPFS)
		peerid, err := peer.IDB58Decode(pid)
		if err != nil {
			glog.Error(err)
		}
		targetPeerAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", pid))
		targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
		node.Peerstore().AddAddr(peerid, targetAddr, peerstore.PermanentAddrTTL)

		s, err := node.NewStream(ctx, peerid, "/echo/1.0.0")
		_, err = s.Write([]byte("Hello, world!\n"))
		if err != nil {
			glog.Error(err)
		}

		out, err := ioutil.ReadAll(s)
		if err != nil {
			glog.Error(err)
		}
		fmt.Printf("read reply: %q\n", out)
	}
	glog.Flush()

}

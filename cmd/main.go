package main

import (
    "os"
    "flag"
    "fmt"
    "time"
	"context"
    //"path/filepath"
    //"strings"
	//"bufio"
	//"sync"
	//"crypto/rand"
	//"github.com/spf13/viper"
	"github.com/golang/glog"
	//"github.com/libp2p/go-libp2p"
	//"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	//"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-core/protocol"
	//"github.com/libp2p/go-libp2p-kad-dht/dual"
	//dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-discovery"
	//maddr "github.com/multiformats/go-multiaddr"
	//p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
    peer "github.com/libp2p/go-libp2p-core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/huo-ju/quorum/internal/pkg/cli"
    "github.com/huo-ju/quorum/internal/pkg/utils"
    "github.com/huo-ju/quorum/internal/pkg/p2p"
    "github.com/huo-ju/quorum/internal/pkg/storage"
    localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
)

var sub *pubsub.Subscription
//var ps *pubsub.PubSub
var ShareTopic string
var node *p2p.Node
var newnode *p2p.Node

// isolates the complex initialization steps
//func constructPeerHost(ctx context.Context, id peer.ID, ps peerstore.Peerstore, options ...libp2p.Option) (host.Host, error) {
//	pkey := ps.PrivKey(id)
//	if pkey == nil {
//		return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
//	}
//	options = append([]libp2p.Option{libp2p.Identity(pkey), libp2p.Peerstore(ps)}, options...)
//	return libp2p.New(ctx, options...)
//}

func handleStream(stream network.Stream) {
	glog.Infof("Got a new stream %s", stream)
}

func mainRet(config cli.Config) int {
    //IFPS soruce note:
    //https://github.com/ipfs/go-ipfs/blob/78c6dba9cc584c5f94d3c610ee95b57272df891f/cmd/ipfs/daemon.go#L360
    //node, err := core.NewNode(req.Context, ncfg)
    //https://github.com/ipfs/go-ipfs/blob/8e6358a4fac40577950260d0c7a7a5d57f4e90a9/core/builder.go#L27
    //ipfs: use fx to build an IPFS node https://github.com/uber-go/fx 
    //node.IPFS(ctx, cfg): https://github.com/ipfs/go-ipfs/blob/7588a6a52a789fa951e1c4916cee5c7a304912c2/core/node/groups.go#L307
    ShareTopic = "test_topic"
	ctx := context.Background()
    if config.IsBootstrap == true {
	    keys,_ := localcrypto.LoadKeys("bootstrap")
        peerid, err := peer.IDFromPublicKey(keys.PubKey)
        if err != nil{
            fmt.Println(err)
        }
        glog.Infof("Your p2p peer ID: %s", peerid)
        listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
        node, err = p2p.NewNode(ctx, keys.PrivKey, listenaddresses, config.JsonTracer)
        fmt.Println(err)
	    glog.Infof("Host created. We are: %s", node.Host.ID())
	    glog.Infof("%s", node.Host.Addrs())

    } else {
	    keys,_ := localcrypto.LoadKeys(config.PeerName)
        peerid, err := peer.IDFromPublicKey(keys.PubKey)
        if err != nil{
            fmt.Println(err)
        }
        glog.Infof("Your p2p peer ID: %s", peerid)

        listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
        newnode, err = p2p.NewNode(ctx, keys.PrivKey, listenaddresses, config.JsonTracer)
		newnode.Host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

        topic, err := newnode.Pubsub.Join(ShareTopic)
        fmt.Println("=======")
        fmt.Printf("topic type %T\n", topic)
        if err != nil {
            fmt.Println("join err")
            fmt.Println(err)
	    }
        _ = newnode.Bootstrap(config)

        glog.Infof("Announcing ourselves...")
        fmt.Println(newnode.RoutingDiscovery)
	    discovery.Advertise(ctx, newnode.RoutingDiscovery, config.RendezvousString)
	    glog.Infof("Successfully announced!")
        //fmt.Println(next)
        //fmt.Println(err)
	    //time.Sleep(time.Second * 5)
        //fmt.Println("Lan Routing Table:")
	    //ddht.LAN.RoutingTable().Print()
        //fmt.Println("Wan Routing Table:")
	    //ddht.WAN.RoutingTable().Print()

        //TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
        count, err := newnode.ConnectPeers(config)

        if count <= 1 {
            for {
		        peers, _:= newnode.FindPeers(config.RendezvousString)
		        if len(peers)>1 { // //connect 2 nodes at least
		            break
		        }
	            time.Sleep(time.Second * 5)
		    }
        }
		//OK we can Subscribe and Publish
        sub, err = topic.Subscribe()
        if err != nil {
            fmt.Println("sub err")
            fmt.Println(err)
	    }
        //storage sync ,publish
        //newnode.EnsureConnect(config.RendezvousString, func(){
        //    fmt.Println("ok connected")
        //})

        //fmt.Println("sub: ")
        //fmt.Println(sub)
        go readFromNetworkLoop(ctx) //start loop to read the subscrbe topic
        go syncDataTicker(config, ctx, topic)
        //err = topic.Publish(ctx, []byte("the message. from: "+config.PeerName))
        //if err != nil {
        //    fmt.Println("publish err")
        //    fmt.Println(err)
	    //} else {
        //    fmt.Println("publish message success")
        //}
    }

	select {}

    return 0
}

func readFromNetworkLoop(ctx context.Context) {
    fmt.Println("run readloop")
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
            fmt.Println(err)
			return
		}
        // save to disk
        fmt.Println(msg)
	}
}

func syncDataTicker(config cli.Config, ctx context.Context, topic *pubsub.Topic){
    fmt.Println("run syncDataTicker")
    syncdataTicker := time.NewTicker(time.Second*30)
	for {
        select {
		    case <-syncdataTicker.C:
                fmt.Println("ticker!")
                newnode.EnsureConnect(config.RendezvousString, func(){
                    storage.JsonSyncData(ctx, "data"+"/"+config.PeerName, topic)
                })
        }
    }
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")
	config, err := cli.ParseFlags()
	if err != nil {
		panic(err)
	}
	if *help {
		fmt.Println("Output a help ")
		fmt.Println()
		fmt.Println("Usage:...")
		flag.PrintDefaults()
		return
	}

	if *version {
        fmt.Println("1.0.0")
        return
    }
	os.Exit(mainRet(config))
}

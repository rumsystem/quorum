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
	"sync"
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
    localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
)

//type Keys struct{
//	PrivKey p2pcrypto.PrivKey
//	PubKey p2pcrypto.PubKey
//}

var sub *pubsub.Subscription
var ShareTopic string
var node *p2p.Node

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
        node, err = p2p.NewNode(ctx, keys.PrivKey, listenaddresses)
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
	    //var ddht *dual.DHT
	    //var routingDiscovery *discovery.RoutingDiscovery
	    //identity := libp2p.Identity(keys.PrivKey)
        //routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
        //    var err error
        //    ddht, err = dual.New(ctx, host)
        //    routingDiscovery = discovery.NewRoutingDiscovery(ddht)
        //    return ddht, err
        //})

        listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
        node, err = p2p.NewNode(ctx, keys.PrivKey, listenaddresses)
	    //host, err := libp2p.New(ctx,
	    //    routing,
        //    libp2p.ListenAddrs(addresses...),
	    //    identity,
	    //)
		node.Host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

        //ps, err = pubsub.NewGossipSub(ctx, node.Host)
        //if err !=nil {
        //fmt.Println("gossip err")
        //fmt.Println(err)
        //}
        topic, err := node.Pubsub.Join(ShareTopic)
        if err != nil {
            fmt.Println("join err")
            fmt.Println(err)
	    }
        sub, err = topic.Subscribe()
        if err != nil {
            fmt.Println("sub err")
            fmt.Println(err)
	    }

        //TOFIX: for test
        //config.BootstrapPeers = dht.DefaultBootstrapPeers
	    var wg sync.WaitGroup
	    for _, peerAddr := range config.BootstrapPeers {
		    peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		    wg.Add(1)
		    go func() {
			    defer wg.Done()
			    if err := node.Host.Connect(ctx, *peerinfo); err != nil {
                    glog.Warning(err)
			    } else {
                    glog.Infof("Connection established with bootstrap node %s:", *peerinfo)
			    }
		    }()
	    }
	    wg.Wait()
        glog.Infof("Announcing ourselves...")
	    discovery.Advertise(ctx, node.RoutingDiscovery, config.RendezvousString)
	    glog.Infof("Successfully announced!")
        //fmt.Println(next)
        //fmt.Println(err)
	    //time.Sleep(time.Second * 5)
        //fmt.Println("Lan Routing Table:")
	    //ddht.LAN.RoutingTable().Print()
        //fmt.Println("Wan Routing Table:")
	    //ddht.WAN.RoutingTable().Print()

	    pctx, _ := context.WithTimeout(ctx, time.Second*10)
	    glog.Infof("find peers with Rendezvous %s ", config.RendezvousString)
        //TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
	    peers, err := discovery.FindPeers(pctx, node.RoutingDiscovery, config.RendezvousString)
	    if err != nil {
	        panic(err)
	    }


        fmt.Println("peers:")
        fmt.Println(peers)
	    for _, peer := range peers {
		    if peer.ID == node.Host.ID() {
		        continue
		    }
		    glog.Infof("Found peer: %s", peer)
            err := node.Host.Connect(ctx, peer)
            if err != nil {
                fmt.Println("====connect error")
                fmt.Println(err)
            }else {
                fmt.Printf("connect: %s \n", peer)
            }
        }

        fmt.Println("sub: ")
        fmt.Println(sub)
        go readLoop(ctx) //start loop to read the subscrbe topic
        go ticker()
        err = topic.Publish(ctx, []byte("the message. from: "+config.PeerName))
        if err != nil {
            fmt.Println("publish err")
            fmt.Println(err)
	    } else {
            fmt.Println("publish message success")
        }

    }

	select {}

    return 0
}

func readLoop(ctx context.Context) {
    fmt.Println("run readloop")
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
            fmt.Println(err)
			return
		}
        fmt.Println(msg)
	}
}

func ticker(){
    fmt.Println("run ticker")
    peerRefreshTicker := time.NewTicker(time.Second*30)
	for {
        select {
		    case <-peerRefreshTicker.C:
                fmt.Println("ticker!")
                idlist := node.Pubsub.ListPeers(ShareTopic)
                fmt.Println(idlist)
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

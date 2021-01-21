package main

import (
    "os"
    "flag"
    "fmt"
    "time"
	"context"
    "path/filepath"
    "strings"
	//"bufio"
	"sync"
	"crypto/rand"
	"github.com/spf13/viper"
	"github.com/golang/glog"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	//dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-discovery"
	maddr "github.com/multiformats/go-multiaddr"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
    peer "github.com/libp2p/go-libp2p-core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type addrList []maddr.Multiaddr
type Config struct {
	RendezvousString string
	BootstrapPeers   addrList
	ListenAddresses  string
	ProtocolID       string
    IsBootstrap     bool
    PeerName        string
}

type Keys struct{
	PrivKey p2pcrypto.PrivKey
	PubKey p2pcrypto.PubKey
}

var sub *pubsub.Subscription
var ps *pubsub.PubSub
var ShareTopic string

func ParseFlags() (Config,error) {
    config := Config{ProtocolID:"/quorum/1.0.0"}
	flag.StringVar(&config.RendezvousString, "rendezvous", "some unique string",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.StringVar(&config.ListenAddresses, "listen", "/ip4/127.0.0.1/tcp/4215", "Adds a multiaddress to the listen list")
	flag.StringVar(&config.PeerName, "peername", "peer", "peername")
    flag.BoolVar(&config.IsBootstrap, "bootstrap", false, "run a bootstrap node")
	flag.Parse()

	return config, nil
}

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
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

// isolates the complex initialization steps
//func constructPeerHost(ctx context.Context, id peer.ID, ps peerstore.Peerstore, options ...libp2p.Option) (host.Host, error) {
//	pkey := ps.PrivKey(id)
//	if pkey == nil {
//		return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
//	}
//	options = append([]libp2p.Option{libp2p.Identity(pkey), libp2p.Peerstore(ps)}, options...)
//	return libp2p.New(ctx, options...)
//}

func writekeysToconfig(priv p2pcrypto.PrivKey, pub p2pcrypto.PubKey) error{

	privkeybytes,err := p2pcrypto.MarshalPrivateKey(priv)
    if err != nil {
        return err
    }
	pubkeybytes,err := p2pcrypto.MarshalPublicKey(pub)
    if err != nil {
        return err
    }
    viper.Set("priv",p2pcrypto.ConfigEncodeKey(privkeybytes))
    viper.Set("pub",p2pcrypto.ConfigEncodeKey(pubkeybytes))
    viper.SafeWriteConfig()
    return nil
}

func loadKeys(keyname string) (*Keys,error){
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.SetConfigName(keyname+"_keys")
	viper.SetConfigType("toml")
    err := viper.ReadInConfig()
    if err != nil {

	    glog.Infof("Keys files not found, generating new keypair..")
	    priv, pub, err := p2pcrypto.GenerateKeyPairWithReader(p2pcrypto.RSA, 4096, rand.Reader)
        if err != nil{
            return nil, err
        }
        writekeysToconfig(priv, pub)
    }
    err = viper.ReadInConfig()
    if err !=nil {
        return nil, err
    }

    privstr := viper.GetString("priv")
    pubstr := viper.GetString("pub")
	glog.Infof("Load keys from config")

    serializedpub,_ := p2pcrypto.ConfigDecodeKey(pubstr)
    pubfromconfig, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
    if err !=nil {
        return nil, err
    }

    serializedpriv,_ := p2pcrypto.ConfigDecodeKey(privstr)
    privfromconfig, err := p2pcrypto.UnmarshalPrivateKey(serializedpriv)
    if err !=nil {
        return nil, err
    }
	return &Keys{PrivKey: privfromconfig, PubKey: pubfromconfig}, nil
}

func handleStream(stream network.Stream) {
	glog.Infof("Got a new stream %s", stream)
}

func mainRet(config Config) int {
    //IFPS soruce note:
    //https://github.com/ipfs/go-ipfs/blob/78c6dba9cc584c5f94d3c610ee95b57272df891f/cmd/ipfs/daemon.go#L360
    //node, err := core.NewNode(req.Context, ncfg)
    //https://github.com/ipfs/go-ipfs/blob/8e6358a4fac40577950260d0c7a7a5d57f4e90a9/core/builder.go#L27
    //ipfs: use fx to build an IPFS node https://github.com/uber-go/fx 
    //node.IPFS(ctx, cfg): https://github.com/ipfs/go-ipfs/blob/7588a6a52a789fa951e1c4916cee5c7a304912c2/core/node/groups.go#L307
    ShareTopic = "test_topic"
	ctx := context.Background()
	fmt.Println(ctx)
    if config.IsBootstrap == true {
	    keys,_ := loadKeys("bootstrap")
        peerid, err := peer.IDFromPublicKey(keys.PubKey)
        if err != nil{
            fmt.Println(err)
        }
        glog.Infof("Your p2p peer ID: %s", peerid)
	    var ddht *dual.DHT
	    var routingDiscovery *discovery.RoutingDiscovery
	    identity := libp2p.Identity(keys.PrivKey)
        routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
            var err error
            ddht, err = dual.New(ctx, host)
            routingDiscovery = discovery.NewRoutingDiscovery(ddht)
            return ddht, err
        })
	    host, err := libp2p.New(ctx,
	        routing,
            libp2p.ListenAddrStrings(config.ListenAddresses),
		    identity,
	    )
        fmt.Println(err)
	    glog.Infof("Host created. We are: %s", host.ID())
	    glog.Infof("%s", host.Addrs())
    } else {
	    keys,_ := loadKeys(config.PeerName)
        peerid, err := peer.IDFromPublicKey(keys.PubKey)
        if err != nil{
            fmt.Println(err)
        }
        glog.Infof("Your p2p peer ID: %s", peerid)
	    var ddht *dual.DHT
	    var routingDiscovery *discovery.RoutingDiscovery
	    identity := libp2p.Identity(keys.PrivKey)
        routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
            var err error
            ddht, err = dual.New(ctx, host)
            routingDiscovery = discovery.NewRoutingDiscovery(ddht)
            return ddht, err
        })

        addresses, _ := StringsToAddrs([]string{config.ListenAddresses})
	    host, err := libp2p.New(ctx,
	        routing,
            libp2p.ListenAddrs(addresses...),
	        identity,
	    )
		host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

        ps, err = pubsub.NewGossipSub(ctx, host)
        if err !=nil {
        fmt.Println("gossip err")
        fmt.Println(err)
        }
        topic, err := ps.Join(ShareTopic)
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
			    if err := host.Connect(ctx, *peerinfo); err != nil {
                    glog.Warning(err)
			    } else {
                    glog.Infof("Connection established with bootstrap node %s:", *peerinfo)
			    }
		    }()
	    }
	    wg.Wait()
        glog.Infof("Announcing ourselves...")
	    discovery.Advertise(ctx, routingDiscovery, config.RendezvousString)
	    glog.Infof("Successfully announced!")
        //fmt.Println(next)
        //fmt.Println(err)
	    time.Sleep(time.Second * 5)
        fmt.Println("Lan Routing Table:")
	    ddht.LAN.RoutingTable().Print()
        fmt.Println("Wan Routing Table:")
	    ddht.WAN.RoutingTable().Print()

	    pctx, _ := context.WithTimeout(ctx, time.Second*10)
	    glog.Infof("find peers with Rendezvous %s ", config.RendezvousString)
        //TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
	    peers, err := discovery.FindPeers(pctx, routingDiscovery, config.RendezvousString)
	    if err != nil {
	        panic(err)
	    }


        fmt.Println("peers:")
        fmt.Println(peers)
	    for _, peer := range peers {
		    if peer.ID == host.ID() {
		        continue
		    }
		    glog.Infof("Found peer: %s", peer)
            err := host.Connect(ctx, peer)
            if err != nil {
                fmt.Println("====connect error")
                fmt.Println(err)
            }else {
                fmt.Printf("connect: %s \n", peer)
            }
			//stream, _ := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))
            //_ = bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	        ////go readData(rw)
	        ////go writeData(rw)
            //glog.Infof("create stream with peer and protocol :%s", config.ProtocolID)

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
                idlist := ps.ListPeers(ShareTopic)
                fmt.Println(idlist)
        }
    }
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")
	config, err := ParseFlags()
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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	//"io"
	//"io/ioutil"
	//"strings"
	//"bufio"
	"context"
	//"math/rand"
	"encoding/json"
	"github.com/golang/glog"
	golog "github.com/ipfs/go-log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-discovery"
	"google.golang.org/protobuf/proto"
	//"github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	msgio "github.com/libp2p/go-msgio"
	//ds "github.com/ipfs/go-datastore"
	blockstore "github.com/huo-ju/go-ipfs-blockstore"
	blocks "github.com/ipfs/go-block-format"
	dsquery "github.com/ipfs/go-datastore/query"
	ds_sync "github.com/ipfs/go-datastore/sync"
	ds_badger "github.com/ipfs/go-ds-badger"
	//"github.com/libp2p/go-msgio"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	//cid "github.com/ipfs/go-cid"
	"github.com/huo-ju/quorum/internal/pkg/api"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/data"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/huo-ju/quorum/internal/pkg/storage"
	"github.com/huo-ju/quorum/internal/pkg/utils"
)

var sub *pubsub.Subscription

//var ps *pubsub.PubSub
var ShareTopic string
var node *p2p.Node
var newnode *p2p.Node

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
		keys, _ := localcrypto.LoadKeys("bootstrap")
		peerid, err := peer.IDFromPublicKey(keys.PubKey)
		if err != nil {
			fmt.Println(err)
		}
		glog.Infof("Your p2p peer ID: %s", peerid)
		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		node, err = p2p.NewNode(ctx, keys.PrivKey, nil, listenaddresses, config.JsonTracer)
		fmt.Println(err)
		glog.Infof("Host created. We are: %s", node.Host.ID())
		glog.Infof("%s", node.Host.Addrs())

	} else {
		keys, _ := localcrypto.LoadKeys(config.PeerName)
		peerid, err := peer.IDFromPublicKey(keys.PubKey)
		if err != nil {
			fmt.Println(err)
		}
		glog.Infof("Your p2p peer ID: %s", peerid)

		blockpath := "data" + "/" + config.PeerName + "_blocks"

		badgerstorage, err := ds_badger.NewDatastore(blockpath, nil)
		bs := blockstore.NewBlockstore(ds_sync.MutexWrap(badgerstorage), "data")

		//set default prefix: /blocks
		//https://github.com/ipfs/go-ipfs-blockstore/blob/fb07d7bc5aece18c62603f36ac02db2e853cadfa/blockstore.go#L24
		//FOR TEST
		query := dsquery.Query{
			//Prefix: "/blocks",
			//KeysOnly: true,
			Limit: 10,
		}
		res, err := badgerstorage.Query(query)
		fmt.Println("====query")
		fmt.Println(query)

		actualE, err := res.Rest()
		if err != nil {
			fmt.Println(err)
		}

		actual := make([]string, len(actualE))
		for _, e := range actualE {
			fmt.Println("e.Key:")
			fmt.Println(e.Key)
			block := blocks.NewBlock(e.Value)
			cid := block.Cid()
			fmt.Println("====cid:")
			fmt.Println(cid)
			k := dshelp.MultihashToDsKey(block.Cid().Hash())
			fmt.Println("====dskey:")
			fmt.Println(k)
		}
		fmt.Println(actual)

		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		newnode, err = p2p.NewNode(ctx, keys.PrivKey, bs, listenaddresses, config.JsonTracer)
		newnode.Host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

		topic, err := newnode.Pubsub.Join(ShareTopic)
		if err != nil {
			fmt.Println("join err")
			fmt.Println(err)
		}
		_ = newnode.Bootstrap(config)

		glog.Infof("Announcing ourselves...")
		fmt.Println(newnode.RoutingDiscovery)
		discovery.Advertise(ctx, newnode.RoutingDiscovery, config.RendezvousString)
		glog.Infof("Successfully announced!")

		//network := bsnet.NewFromIpfsHost(host, routingDiscovery)
		//exchange := bitswap.New(ctx, network, bstore)
		relationdb := "a_block_relation_db_object"
		askheadservice := p2p.NewHeadBlockService(newnode.Host, relationdb)
		fmt.Println("register askheadservice")
		fmt.Println(askheadservice)

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
			//for {
			//	peers, _ := newnode.FindPeers(config.RendezvousString)
			//	if len(peers) > 1 { // //connect 2 nodes at least
			//		break
			//	}
			//	time.Sleep(time.Second * 5)
			//}
		}
		//OK we can Subscribe and Publish
		sub, err = topic.Subscribe()
		if err != nil {
			fmt.Println("sub err")
			fmt.Println(err)
		}
		go readFromNetworkLoop(ctx, config, bs) //start loop to read the subscrbe topic
		go AskHeadBlockID(config, ctx)
		//go syncDataTicker(config, ctx, topic)
		//run local http api service
		h := &api.Handler{Node: newnode, PubsubTopic: topic, Ctx: ctx}
		go StartAPIServer(config, h)

	}

	select {}
	return 0
}

func readFromNetworkLoop(ctx context.Context, config cli.Config, bs blockstore.Blockstore) {
	fmt.Println("run readloop")
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		//storage.WriteJsonToFile("data"+"/"+config.PeerName,  msg.Data)
		fmt.Printf("receive msg: %T\n", msg)
		//verify msg

		var activity data.Activity
		err = json.Unmarshal(msg.Data, &activity)

		if err != nil {
			fmt.Println(err)
		} else {
			block := blocks.NewBlock(msg.Data)
			cid := block.Cid()
			fmt.Printf("receive cid: %s \n ", cid)
			err = bs.Put(block)
			fmt.Println(err)
		}
	}
}

//func RandBuf(r *rand.Rand, length int) []byte {
//	buf := make([]byte, length)
//	for i := range buf {
//		buf[i] = byte(r.Intn(256))
//	}
//	return buf[:]
//}

func AskHeadBlockID(config cli.Config, ctx context.Context) {
	syncdataTicker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-syncdataTicker.C:
			glog.Infof("Run ask head block")
			peers, _ := newnode.FindPeers(config.RendezvousString)
			for _, peer := range peers {

				if newnode.PeerID != peer.ID { //peer is not myself
					glog.Infof("Open a new stream to  %s", peer.ID)
					s, err := newnode.Host.NewStream(ctx, peer.ID, p2p.HeadBlockProtocolID)
					if err != nil {
						glog.Errorf("Open new stream to %s err: %s", peer.ID, err)
					} else {

						askheadmsg := &quorumpb.BlockMessage{
							Type:  quorumpb.BlockMessage_ASKHEAD,
							Value: "",
						}

						pbmsg, err := proto.Marshal(askheadmsg)
						if err != nil {
							glog.Errorf("Marshal askheadmsg err: %s", err)
						} else {
							mrw := msgio.NewReadWriter(s)
							err = mrw.WriteMsg(pbmsg)
							if err != nil {
								glog.Errorf("Write ASKHEAD message err: %s", err)
							} else { //read reply message
								glog.Infof("Send ASKHEAD msg to : %s", peer.ID)
								go func() { //TOFIX: set a timeout flag to avoid long loop
									for {
										msg, err := mrw.ReadMsg()
										if err == nil {
											pb := &quorumpb.BlockMessage{}
											err = proto.Unmarshal(msg, pb)
											mrw.ReleaseMsg(msg)
											glog.Infof("ASKHEAD reply : %s", string(msg))
											s.Close()
											return
										} else {
											glog.Errorf("read ASKHEAD reply err: %s", err)
											s.Reset()
										}
									}
								}()
							}
						}
					}
				}
			}
		}
	}
}

func syncDataTicker(config cli.Config, ctx context.Context, topic *pubsub.Topic) {
	fmt.Println("run syncDataTicker")
	syncdataTicker := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-syncdataTicker.C:
			fmt.Println("ticker!")
			newnode.EnsureConnect(config.RendezvousString, func() {
				storage.JsonSyncData(ctx, "data"+"/"+config.PeerName, topic)

				block := blocks.NewBlock([]byte("some data1"))
				cid := block.Cid()
				fmt.Println("cid:")
				fmt.Println(cid)
				fmt.Println("exchange:")
				fmt.Println(newnode.Exchange)
				readblock, err := newnode.Exchange.GetBlock(ctx, cid)
				fmt.Println("peer:" + config.PeerName)
				fmt.Println(readblock)
				fmt.Println(err)
			})
		}
	}
}

//StartAPIServer : Start the web server
func StartAPIServer(config cli.Config, h *api.Handler) {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	r := e.Group("/api")
	r.POST("/create", h.Create)
	r.GET("/info", h.Info)
	e.Logger.Fatal(e.Start(config.APIListenAddresses))
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")
	config, err := cli.ParseFlags()
	lvl, err := golog.LevelFromString("info")
	if err != nil {
		panic(err)
	}
	golog.SetAllLoggers(lvl)
	golog.SetLogLevel("pubsub", "debug")

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

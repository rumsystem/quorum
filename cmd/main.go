package main

import (
	"context"
	//"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	//"time"

	"github.com/golang/glog"
	golog "github.com/ipfs/go-log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-discovery"

	//msgio "github.com/libp2p/go-msgio"
	//"google.golang.org/protobuf/proto"

	"github.com/huo-ju/quorum/internal/pkg/api"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"

	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	//"github.com/huo-ju/quorum/internal/pkg/data"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	//quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	//"github.com/huo-ju/quorum/internal/pkg/storage"
	"github.com/huo-ju/quorum/internal/pkg/utils"
	//blocks "github.com/ipfs/go-block-format"
	//pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/huo-ju/quorum/internal/pkg/cli"
)

const PUBLIC_CHANNEL = "all_node_public_channel"

var node *p2p.Node

func handleStream(stream network.Stream) {
	glog.Infof("Got a new stream <%s>", stream)
}

func mainRet(config cli.Config) int {

	//Initial chain context
	chain.InitContext()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if config.IsBootstrap == true {
		keys, _ := localcrypto.LoadKeys("bootstrap")

		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		node, err := p2p.NewNode(ctx, keys.PrivKey, nil, listenaddresses, config.JsonTracer)

		if err != nil {
			glog.Fatalf(err.Error())
			return 0
		}

		glog.Infof("Host created, ID:<%s>, Address:<%s>", node.Host.ID(), node.Host.Addrs())
		chain.GetContext().Privatekey = keys.PrivKey
		chain.GetContext().PublicKey = keys.PubKey
		chain.GetContext().PeerId = node.Host.ID()

	} else {

		keys, _ := localcrypto.LoadKeys(config.PeerName)
		peerid, err := peer.IDFromPublicKey(keys.PubKey)
		if err != nil {
			glog.Fatalf(err.Error())
			cancel()
			return 0
		}

		glog.Infof("peer_id created, <%s>", peerid)

		datapath := "data" + "/" + config.PeerName

		chain.GetContext().Privatekey = keys.PrivKey
		chain.GetContext().PublicKey = keys.PubKey
		chain.GetContext().PeerId = peerid
		chain.GetContext().InitDB(datapath)

		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		node, err = p2p.NewNode(ctx, keys.PrivKey, chain.GetContext().BlockStorage, listenaddresses, config.JsonTracer)
		node.Host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

		_ = node.Bootstrap(ctx, config)

		glog.Infof("Announcing ourselves...")

		discovery.Advertise(ctx, node.RoutingDiscovery, config.RendezvousString)
		glog.Infof("Successfully announced!")

		//TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
		count, err := node.ConnectPeers(ctx, config)

		if err != nil {
			glog.Fatalf(err.Error())
			return 0
		} else if count <= 1 {
			//for {
			//	peers, _ := node.FindPeers(config.RendezvousString)
			//	if len(peers) > 1 { // //connect 2 nodes at least
			//		break
			//	}
			//	time.Sleep(time.Second * 5)
			//}
			glog.Errorf("can not connec to other peer, maybe I am the first one?")
		}

		//join public channel
		err = chain.GetContext().JoinPublicChannel(node, PUBLIC_CHANNEL, ctx, config)
		if err != nil {
			return 0
		}

		//load group from localdb

		//test only
		err = chain.JoinTestGroup()
		if err != nil {
			return 0
		}

		//run local http api service
		h := &api.Handler{Node: node, ChainCtx: chain.GetContext(), Ctx: ctx}
		go StartAPIServer(config, h)

	}

	//attach signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-ch
	signal.Stop(ch)

	//cleanup before exit
	glog.Infof("On Signal <%s>", signalType)
	glog.Infof("Exit command received. Exiting...")

	return 0
}

//StartAPIServer : Start local web server
func StartAPIServer(config cli.Config, h *api.Handler) {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	r := e.Group("/api")
	//r.POST("/create", h.Create)
	r.POST("/posttogroup", h.PostToGroup)
	r.POST("/addgroup", h.AddGroup)
	r.POST("/addgroupuser", h.AddGroupUser)
	r.POST("/rmgroup", h.RmGroup)
	r.POST("/rmgroupuser", h.RmGroupUser)
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

	if config.IsDebug == true {
		golog.SetAllLoggers(lvl)
		golog.SetLogLevel("pubsub", "debug")
		golog.SetLogLevel("autonat", "debug")
	}

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

/* code from huo

import

//dsquery "github.com/ipfs/go-datastore/query"
//ds_sync "github.com/ipfs/go-datastore/sync"
//ds_badger "github.com/ipfs/go-ds-badger"
//dshelp "github.com/ipfs/go-ipfs-ds-help"
//cid "github.com/ipfs/go-cid"
//"github.com/libp2p/go-msgio"
//ds "github.com/ipfs/go-datastore"
//"github.com/libp2p/go-libp2p-core/host"


mainret()

//set default prefix: /blocks
//https://github.com/ipfs/go-ipfs-blockstore/blob/fb07d7bc5aece18c62603f36ac02db2e853cadfa/blockstore.go#L24
//FOR TEST

	query := dsquery.Query{
		//Prefix: "/blocks",
		//KeysOnly: true,
		Limit: 10,
	}
	res, err := badgerstorage.Query(query)
	glog.Infof("====query")
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

fmt.Println(node.RoutingDiscovery)

//network := bsnet.NewFromIpfsHost(host, routingDiscovery)
//exchange := bitswap.New(ctx, network, bstore)

//commented by cuim
//relationdb := "a_block_relation_db_object"
//askheadservice := p2p.NewHeadBlockService(node.Host, relationdb)
//glog.Infof("Register askheadservice")

//fmt.Println(askheadservice)

//fmt.Println(next)
//fmt.Println(err)
//time.Sleep(time.Second * 5)
//fmt.Println("Lan Routing Table:")
//ddht.LAN.RoutingTable().Print()
//fmt.Println("Wan Routing Table:")
//ddht.WAN.RoutingTable().Print()

//commented by cuim
//go readFromNetworkLoop(ctx, config, bs) //start loop to read the subscrbe topic
//go AskHeadBlockID(config, ctx)
//go syncDataTicker(config, ctx, topic)

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

func syncDataTicker(config cli.Config, ctx context.Context, topic *pubsub.Topic) {
	fmt.Println("run syncDataTicker")
	syncdataTicker := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-syncdataTicker.C:
			fmt.Println("ticker!")
			node.EnsureConnect(ctx, config.RendezvousString, func() {
				storage.JsonSyncData(ctx, "data"+"/"+config.PeerName, topic)

				block := blocks.NewBlock([]byte("some data1"))
				cid := block.Cid()
				fmt.Println("cid:")
				fmt.Println(cid)
				fmt.Println("exchange:")
				fmt.Println(node.Exchange)
				readblock, err := node.Exchange.GetBlock(ctx, cid)
				fmt.Println("peer:" + config.PeerName)
				fmt.Println(readblock)
				fmt.Println(err)
			})
		}
	}
}

func AskHeadBlockID(config cli.Config, ctx context.Context) {
	syncdataTicker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-syncdataTicker.C:
			glog.Infof("Run ask head block")

			glog.Infof("We are: %s", node.Host.ID())
			glog.Infof("Address: %s", node.Host.Addrs())
			peers, _ := node.FindPeers(ctx, config.RendezvousString)
			for _, peer := range peers {

				if node.PeerID != peer.ID { //peer is not myself
					glog.Infof("Open a new stream to  %s", peer.ID)
					s, err := node.Host.NewStream(ctx, peer.ID, p2p.HeadBlockProtocolID)
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
											return
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

*/

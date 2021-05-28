package main

import (
	"context"
	//"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	//"time"

	"github.com/golang/glog"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	golog "github.com/ipfs/go-log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	//discovery "github.com/libp2p/go-libp2p-discovery"
	"google.golang.org/protobuf/encoding/protojson"

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

//const PUBLIC_CHANNEL = "all_node_public_channel"

var node *p2p.Node

func handleStream(stream network.Stream) {
	glog.Infof("Got a new stream <%s>", stream)
}

func mainRet(config cli.Config) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if config.IsBootstrap == true {
		keys, _ := localcrypto.LoadKeys(config.ConfigDir, "bootstrap")

		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		//bootstrop node connections: low watermarks: 1000  hi watermarks 50000, grace 30s
		node, err := p2p.NewNode(ctx, config.IsBootstrap, keys.PrivKey, connmgr.NewConnManager(1000, 50000, 30), listenaddresses, config.JsonTracer)

		if err != nil {
			glog.Fatalf(err.Error())
			return 0
		}

		glog.Infof("Host created, ID:<%s>, Address:<%s>", node.Host.ID(), node.Host.Addrs())
		h := &api.Handler{}
		go StartAPIServer(config, h, true)
	} else {
		keys, _ := localcrypto.LoadKeys(config.ConfigDir, config.PeerName)
		peerid, err := peer.IDFromPublicKey(keys.PubKey)
		if err != nil {
			glog.Fatalf(err.Error())
			cancel()
			return 0
		}

		glog.Infof("peer_id created, <%s>", peerid)

		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		//normal node connections: low watermarks: 10  hi watermarks 200, grace 60s
		node, err = p2p.NewNode(ctx, config.IsBootstrap, keys.PrivKey, connmgr.NewConnManager(10, 200, 60), listenaddresses, config.JsonTracer)
		node.Host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

		_ = node.Bootstrap(ctx, config)

		for _, addr := range node.Host.Addrs() {
			p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.Host.ID())
			glog.Infof("Peer ID:<%s>, Peer Address:<%s>", node.Host.ID(), p2paddr)
		}

		//Discovery and Advertise had been replaced by PeerExchange
		//glog.Infof("Announcing ourselves...")
		//discovery.Advertise(ctx, node.RoutingDiscovery, config.RendezvousString)
		//glog.Infof("Successfully announced!")

		//peerok := make(chan struct{})
		//go node.ConnectPeers(ctx, peerok, config)

		//select {
		//case <-peerok:
		//	glog.Infof("Connected to enough peers.")
		//}

		datapath := config.DataDir + "/" + config.PeerName
		chain.InitCtx(ctx, node, datapath)
		chain.GetChainCtx().Privatekey = keys.PrivKey
		chain.GetChainCtx().PublicKey = keys.PubKey
		chain.GetChainCtx().PeerId = peerid

		//join public channel
		//err = chain.GetChainCtx().JoinPublicChannel(node, PUBLIC_CHANNEL, ctx, config)
		//if err != nil {
		//	return 0
		//}

		err = chain.GetChainCtx().SyncAllGroup()
		if err != nil {
			glog.Fatalf(err.Error())
			return 0
		}

		//run local http api service
		h := &api.Handler{Node: node, ChainCtx: chain.GetChainCtx(), Ctx: ctx}
		go StartAPIServer(config, h, false)
	}

	//attach signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-ch
	signal.Stop(ch)

	if config.IsBootstrap != true {
		err := chain.GetChainCtx().QuitPublicChannel()
		if err != nil {
			return 0
		}
		chain.Release()
	}

	//cleanup before exit
	glog.Infof("On Signal <%s>", signalType)
	glog.Infof("Exit command received. Exiting...")

	return 0
}

type CustomBinder struct{}

func (cb *CustomBinder) Bind(i interface{}, c echo.Context) (err error) {
	db := new(echo.DefaultBinder)
	switch i.(type) {
	case *quorumpb.Activity:
		bodyBytes, err := ioutil.ReadAll(c.Request().Body)
		err = protojson.Unmarshal(bodyBytes, i.(*quorumpb.Activity))
		return err
	default:
		if err = db.Bind(i, c); err != echo.ErrUnsupportedMediaType {
			return
		}
		return err
	}
}

//StartAPIServer : Start local web server
func StartAPIServer(config cli.Config, h *api.Handler, isbootstrapnode bool) {
	e := echo.New()
	e.Binder = new(CustomBinder)
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	r := e.Group("/api")
	if isbootstrapnode == false {
		r.POST("/v1/group", h.CreateGroup)
		r.DELETE("/v1/group", h.RmGroup)
		r.POST("/v1/group/join", h.JoinGroup)
		r.POST("/v1/group/leave", h.LeaveGroup)
		r.POST("/v1/group/content", h.PostToGroup)
		r.GET("/v1/node", h.GetNodeInfo)
		r.GET("/v1/block", h.GetBlock)
		r.GET("/v1/trx", h.GetTrx)
		r.GET("/v1/group/content", h.GetGroupCtn)
		r.GET("/v1/group", h.GetGroups)
		r.GET("/v1/network", h.GetNetwork)
		r.POST("/v1/network/peers", h.AddPeers)
		r.POST("/v1/group/blacklist", h.MgrGrpBlkList)
		r.GET("/v1/group/blacklist", h.GetBlockedUsrList)
	} else {
		r.GET("/v1/node", h.GetBootStropNodeInfo)
	}

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

	if _, err := os.Stat(config.DataDir); err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(config.DataDir, 0755)
			if err != nil {
				panic(err)
			}
		} else {
			if err != nil {
				panic(err)
			}
		}
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

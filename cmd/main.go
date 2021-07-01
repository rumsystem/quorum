package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"syscall"

	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	dsbadger2 "github.com/ipfs/go-ds-badger2"
	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/huo-ju/quorum/internal/pkg/api"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"

	"github.com/huo-ju/quorum/internal/pkg/cli"
	"github.com/huo-ju/quorum/internal/pkg/options"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	"github.com/huo-ju/quorum/internal/pkg/utils"
)

//const PUBLIC_CHANNEL = "all_node_public_channel"

var node *p2p.Node
var signalch chan os.Signal
var mainlog = logging.Logger("main")

func mainRet(config cli.Config) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peername := config.PeerName

	if config.IsBootstrap == true {
		peername = "bootstrap"
	}

	keys, err := localcrypto.LoadKeysFrom(config.ConfigDir, peername, "txt")
	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}
	peerid, err := peer.IDFromPublicKey(keys.PubKey)
	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}

	mainlog.Infof("eth addresss: <%s>", keys.EthAddr)

	ds, err := dsbadger2.NewDatastore(path.Join(config.DataDir, fmt.Sprintf("%s-%s", peername, "peerstore")), &dsbadger2.DefaultOptions)
	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}

	//Load node options
	nodeoptions, err := options.Load(config.ConfigDir, peername)
	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}

	if config.IsBootstrap == true {
		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		//bootstrop node connections: low watermarks: 1000  hi watermarks 50000, grace 30s
		node, err := p2p.NewNode(ctx, nodeoptions, config.IsBootstrap, ds, keys.PrivKey, connmgr.NewConnManager(1000, 50000, 30), listenaddresses, config.JsonTracer)

		if err != nil {
			mainlog.Fatalf(err.Error())
			return 0
		}

		mainlog.Infof("Host created, ID:<%s>, Address:<%s>", node.Host.ID(), node.Host.Addrs())
		h := &api.Handler{}
		go StartAPIServer(config, h, true)
	} else {
		listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})
		//normal node connections: low watermarks: 10  hi watermarks 200, grace 60s
		node, err = p2p.NewNode(ctx, nodeoptions, config.IsBootstrap, ds, keys.PrivKey, connmgr.NewConnManager(10, 200, 60), listenaddresses, config.JsonTracer)
		_ = node.Bootstrap(ctx, config)

		for _, addr := range node.Host.Addrs() {
			p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.Host.ID())
			mainlog.Infof("Peer ID:<%s>, Peer Address:<%s>", node.Host.ID(), p2paddr)
		}

		//Discovery and Advertise had been replaced by PeerExchange
		//mainlog.Infof("Announcing ourselves...")
		//discovery.Advertise(ctx, node.RoutingDiscovery, config.RendezvousString)
		//mainlog.Infof("Successfully announced!")

		//peerok := make(chan struct{})
		//go node.ConnectPeers(ctx, peerok, config)

		//select {
		//case <-peerok:
		//	mainlog.Infof("Connected to enough peers.")
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
			mainlog.Fatalf(err.Error())
			return 0
		}

		//run local http api service
		h := &api.Handler{Node: node, ChainCtx: chain.GetChainCtx(), Ctx: ctx}
		go StartAPIServer(config, h, false)
	}

	//attach signal
	signalch = make(chan os.Signal, 1)
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	if config.IsBootstrap != true {
		err := chain.GetChainCtx().QuitPublicChannel()
		if err != nil {
			return 0
		}
		chain.Release()
	}

	//cleanup before exit
	mainlog.Infof("On Signal <%s>", signalType)
	mainlog.Infof("Exit command received. Exiting...")

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
	r.GET("/quit", quitapp)
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

func quitapp(c echo.Context) (err error) {
	mainlog.Infof("/api/quit has been called, send Signal SIGTERM...")
	signalch <- syscall.SIGTERM
	return nil
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")
	config, err := cli.ParseFlags()
	lvl, err := logging.LevelFromString("info")
	if err != nil {
		panic(err)
	}

	if config.IsDebug == true {
		logging.SetAllLoggers(lvl)
		logging.SetLogLevel("main", "debug")
		logging.SetLogLevel("pubsub", "debug")
		logging.SetLogLevel("autonat", "debug")
		logging.SetLogLevel("chain", "debug")
		logging.SetLogLevel("dbmgr", "debug")
		logging.SetLogLevel("chainctx", "debug")
		logging.SetLogLevel("group", "debug")

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

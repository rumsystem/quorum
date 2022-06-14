package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkapi "github.com/rumsystem/quorum/pkg/nodesdk/api"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

const DEFAUT_KEY_NAME string = "nodesdk_default"

var (
	ReleaseVersion string
	GitCommit      string
	signalch       chan os.Signal
	mainlog        = logging.Logger("nodesdk")
)

func main() {
	if ReleaseVersion == "" {
		ReleaseVersion = "v1.0.0"
	}

	if GitCommit == "" {
		GitCommit = "devel"
	}

	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")

	config, err := cli.ParseFlags()

	lvl, err := logging.LevelFromString("info")
	logging.SetAllLoggers(lvl)

	if err != nil {
		panic(err)
	}

	if config.IsDebug == true {
		logging.SetLogLevel("nodesdk", "debug")
	}

	if *help {
		fmt.Println("Output a help ")
		fmt.Println()
		fmt.Println("Usage:...")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Printf("%s - %s\n", ReleaseVersion, GitCommit)
		return
	}

	if err := utils.EnsureDir(config.DataDir); err != nil {
		panic(err)
	}

	_, _, err = utils.NewTLSCert()
	if err != nil {
		panic(err)
	}

	os.Exit(mainRet(config))
}

func mainRet(config cli.Config) int {
	signalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mainlog.Infof("Version: %s", GitCommit)
	peername := config.PeerName

	//Load node options
	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	ks, defaultkey, err := InitDefaultKeystore(config, nodeoptions)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	keys, err := localcrypto.SignKeytoPeerKeys(defaultkey)

	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}

	peerid, ethaddr, err := ks.GetPeerInfo(DEFAUT_KEY_NAME)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	mainlog.Infof("eth addresss: <%s>", ethaddr)

	nodename := "nodesdk_default"

	datapath := config.DataDir + "/" + config.PeerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		mainlog.Fatalf(err.Error())
	}

	nodesdkctx.Init(ctx, nodename, dbManager, chainstorage.NewChainStorage(dbManager))
	nodesdkctx.GetCtx().Keystore = ks
	nodesdkctx.GetCtx().PublicKey = keys.PubKey
	nodesdkctx.GetCtx().PeerId = peerid

	//run local http api service
	nodeHandler := &nodesdkapi.NodeSDKHandler{
		NodeSdkCtx: nodesdkctx.GetCtx(),
		Ctx:        ctx,
	}

	nodeApiAddress := "https://%s/api/v1"
	if config.NodeAPIListenAddress[:1] == ":" {
		nodeApiAddress = fmt.Sprintf(nodeApiAddress, "localhost"+config.NodeAPIListenAddress)
	} else {
		nodeApiAddress = fmt.Sprintf(nodeApiAddress, config.NodeAPIListenAddress)
	}

	//start node sdk server
	go nodesdkapi.StartNodeSDKServer(config, signalch, nodeHandler, nodeoptions)

	//attach signal
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	nodesdkctx.GetDbMgr().CloseDb()

	//cleanup before exit
	mainlog.Infof("On Signal <%s>", signalType)
	mainlog.Infof("Exit command received. Exiting...")

	return 0
}

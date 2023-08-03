package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	_ "github.com/golang/protobuf/ptypes/timestamp" //import for swaggo
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/util"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	_ "github.com/multiformats/go-multiaddr" //import for swaggo
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/api"
	appapi "github.com/rumsystem/quorum/pkg/chainapi/appapi"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/spf13/cobra"
)

var (
	rumNodeFlag     = cli.RumNodeFlag{ProtocolID: "/quorum/1.0.0"}
	rumNode         *p2p.Node
	rumNodeSignalch chan os.Signal
)

var rumNodeCmd = &cobra.Command{
	Use:   "rumnode",
	Short: "Run rumnode",
	Run: func(cmd *cobra.Command, args []string) {
		if rumNodeFlag.KeyStorePwd == "" {
			rumNodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		rumNodeFlag.IsDebug = isDebug
		runRumnode(rumNodeFlag)
	},
}

func init() {
	rootCmd.AddCommand(rumNodeCmd)

	flags := rumNodeCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&rumNodeFlag.PeerName, "peername", "peer", "peername")
	flags.StringVar(&rumNodeFlag.ConfigDir, "configdir", "./config/", "config and keys dir")
	flags.StringVar(&rumNodeFlag.DataDir, "datadir", "./data/", "data dir")
	flags.StringVar(&rumNodeFlag.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flags.StringVar(&rumNodeFlag.KeyStoreName, "keystorename", "default", "keystore name")
	flags.StringVar(&rumNodeFlag.KeyStorePwd, "keystorepass", "", "keystore password")
	flags.Var(&rumNodeFlag.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.StringVar(&rumNodeFlag.APIHost, "apihost", "", "Domain or public ip addresses for api server")
	flags.UintVar(&rumNodeFlag.APIPort, "apiport", 5215, "api server listen port")
	flags.StringVar(&rumNodeFlag.CertDir, "certdir", "certs", "ssl certificate directory")
	flags.StringVar(&rumNodeFlag.ZeroAccessKey, "zerosslaccesskey", "", "zerossl access key, get from: https://app.zerossl.com/developer")
	flags.Var(&rumNodeFlag.BootstrapPeers, "peer", "bootstrap peer address")
	flags.StringVar(&rumNodeFlag.SkipPeers, "skippeers", "", "peer id lists, will be skipped in the pubsub connection")
	flags.StringVar(&rumNodeFlag.JsonTracer, "jsontracer", "", "output tracer data to a json file")
	flags.BoolVar(&rumNodeFlag.AutoAck, "autoack", true, "auto ack the transactions in pubqueue")
	flags.BoolVar(&rumNodeFlag.EnableRelay, "autorelay", true, "enable relay")
}

func runRumnode(config cli.RumNodeFlag) {
	// NOTE: hardcode
	const defaultKeyName = "default"

	color.Green("Version: %s", utils.GitCommit)

	rumNodeSignalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peername := config.PeerName

	if err := utils.EnsureDir(config.DataDir); err != nil {
		logger.Fatalf("check or create directory: %s failed: %s", config.DataDir, err)
	}

	//Load node options from config
	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	// overwrite by cli flags
	nodeoptions.EnableRelay = config.EnableRelay

	keystoreParam := InitKeystoreParam{
		KeystoreName:   config.KeyStoreName,
		KeystoreDir:    config.KeyStoreDir,
		KeystorePwd:    config.KeyStorePwd,
		ConfigDir:      config.ConfigDir,
		PeerName:       config.PeerName,
		DefaultKeyName: defaultKeyName,
	}

	ks, defaultkey, err := InitDefaultKeystore(keystoreParam, nodeoptions)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	keys, err := localcrypto.SignKeytoPeerKeys(defaultkey)
	if err != nil {
		logger.Fatalf(err.Error())
		cancel()
	}

	peerid, ethaddr, err := ks.GetPeerInfo(defaultKeyName)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	logger.Infof("eth addresss: <%s>", ethaddr)
	CheckLockError(err)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	nodename := "rumNode_default"

	datapath := config.DataDir + "/" + config.PeerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf(err.Error())
	}
	newchainstorage := chainstorage.NewChainStorage(dbManager)

	//normal node connections: low watermarks: 10  hi watermarks 200, grace 60s
	cm, err := connmgr.NewConnManager(10, nodeoptions.ConnsHi, connmgr.WithGracePeriod(60*time.Second))
	if err != nil {
		logger.Fatalf(err.Error())
	}

	SkipPeerIdList := strings.Split(config.SkipPeers, ",")
	rumNode, err = p2p.NewNode(ctx, nodename, nodeoptions, false, defaultkey, cm, config.ListenAddresses, SkipPeerIdList, config.JsonTracer)
	//rumNode must enable rumexchange for sync block
	if err == nil {
		rumNode.SetRumExchange(ctx)
	}

	for _, addr := range rumNode.Host.Addrs() {
		p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), rumNode.Host.ID())
		logger.Infof("Peer ID:<%s>, Peer Address:<%s>", rumNode.Host.ID(), p2paddr)
	}

	nodectx.InitCtx(ctx, nodename, rumNode, dbManager, newchainstorage, "pubsub", utils.GitCommit, nodectx.RUM_NODE)
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	nodectx.GetNodeCtx().PeerId = peerid

	//initial conn
	conn.InitConn()

	//initial group manager
	chain.InitGroupMgr()
	//if nodeoptions.IsRexTestMode == true {
	//	chain.GetGroupMgr().SetRumExchangeTestMode()
	//}

	//load all groups
	err = chain.GetGroupMgr().LoadAllGroups()
	if err != nil {
		logger.Fatalf(err.Error())
	}

	if err := rumNode.Bootstrap(ctx, config.BootstrapPeers); err != nil {
		logger.Fatal(err)
	}
	//Discovery and Advertise had been replaced by PeerExchange
	logger.Infof("Announcing ourselves...")
	discovery.Advertise(ctx, rumNode.RoutingDiscovery, config.RendezvousString)
	logger.Infof("Successfully announced!")
	peerok := make(chan struct{})
	go rumNode.ConnectPeers(ctx, peerok, nodeoptions.MaxPeers, config.RendezvousString)

	appdb, err := appdata.CreateAppDb(datapath)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	CheckLockError(err)

	// init the websocket manager
	websocketManager := api.NewWebsocketManager()
	go websocketManager.Start()

	//start sync all groups
	err = chain.GetGroupMgr().StartSyncAllGroups()
	if err != nil {
		logger.Fatalf(err.Error())
	}

	//run local http api service
	h := &api.Handler{
		Node:             rumNode,
		NodeCtx:          nodectx.GetNodeCtx(),
		Ctx:              ctx,
		GitCommit:        utils.GitCommit,
		Appdb:            appdb,
		ChainAPIdb:       newchainstorage,
		WebsocketManager: websocketManager,
	}

	apiaddress := fmt.Sprintf("http://localhost:%d/api/v1", config.APIPort)
	appsync := appdata.NewAppSyncAgent(apiaddress, nodectx.GetNodeCtx().Name, appdb, dbManager)
	appsync.Start(10)
	apph := &appapi.Handler{
		Appdb:     appdb,
		Trxdb:     newchainstorage,
		GitCommit: utils.GitCommit,
		Apiroot:   apiaddress,
		ConfigDir: config.ConfigDir,
		PeerName:  config.PeerName,
		NodeName:  nodectx.GetNodeCtx().Name,
	}
	startParam := api.StartServerParam{
		IsDebug:       config.IsDebug,
		APIHost:       config.APIHost,
		APIPort:       config.APIPort,
		CertDir:       config.CertDir,
		ZeroAccessKey: config.ZeroAccessKey,
	}
	go api.StartRumNodeServer(startParam, rumNodeSignalch, h, apph, rumNode, nodeoptions, ks, ethaddr)

	//attach signal
	signal.Notify(rumNodeSignalch, os.Interrupt, syscall.SIGTERM)
	signalType := <-rumNodeSignalch
	signal.Stop(rumNodeSignalch)

	chain.GetGroupMgr().StopSyncAllGroups()
	//teardown all groups
	chain.GetGroupMgr().TeardownAllGroups()
	//close ctx db
	nodectx.GetDbMgr().CloseDb()

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")
}

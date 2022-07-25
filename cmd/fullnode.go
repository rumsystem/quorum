package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	_ "github.com/golang/protobuf/ptypes/timestamp" //import for swaggo
	dsbadger2 "github.com/ipfs/go-ds-badger2"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	discovery "github.com/libp2p/go-libp2p-discovery"
	_ "github.com/multiformats/go-multiaddr" //import for swaggo
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/stats"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/api"
	appapi "github.com/rumsystem/quorum/pkg/chainapi/appapi"
	"github.com/spf13/cobra"
	_ "google.golang.org/protobuf/proto" //import for swaggo
)

var (
	fnodeFlag = cli.FullnodeFlag{ProtocolID: "/quorum/1.0.0"}
	node      *p2p.Node
	signalch  chan os.Signal
)

var fullnodeCmd = &cobra.Command{
	Use:   "fullnode",
	Short: "Run fullnode",
	Run: func(cmd *cobra.Command, args []string) {
		if fnodeFlag.KeyStorePwd == "" {
			fnodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runFullnode(fnodeFlag)
	},
}

func init() {
	rootCmd.AddCommand(fullnodeCmd)

	flags := fullnodeCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&fnodeFlag.PeerName, "peername", "peer", "peername")
	flags.StringVar(&fnodeFlag.ConfigDir, "configdir", "./config/", "config and keys dir")
	flags.StringVar(&fnodeFlag.DataDir, "datadir", "./data/", "config dir")
	flags.StringVar(&fnodeFlag.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flags.StringVar(&fnodeFlag.KeyStoreName, "keystorename", "default", "keystore name")
	flags.StringVar(&fnodeFlag.KeyStorePwd, "keystorepass", "", "keystore password")
	flags.Var(&fnodeFlag.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.StringVar(&fnodeFlag.APIHost, "apihost", "", "Domain or public ip addresses for api server")
	flags.UintVar(&fnodeFlag.APIPort, "apiport", 5215, "api server listen port")
	flags.StringVar(&fnodeFlag.CertDir, "certdir", "certs", "ssl certificate directory")
	flags.StringVar(&fnodeFlag.ZeroAccessKey, "zerosslaccesskey", "", "zerossl access key, get from: https://app.zerossl.com/developer")
	flags.Var(&fnodeFlag.BootstrapPeers, "peer", "bootstrap peer address")
	flags.StringVar(&fnodeFlag.JsonTracer, "jsontracer", "", "output tracer data to a json file")
	flags.BoolVar(&fnodeFlag.IsDebug, "debug", false, "show debug log")
	flags.BoolVar(&fnodeFlag.IsRexTestMode, "rextest", false, "RumExchange Test Mode")
	flags.BoolVar(&fnodeFlag.IsBootstrap, "bootstrap", false, "run a bootstrap node")
	flags.BoolVar(&fnodeFlag.AutoAck, "autoack", false, "auto ack the transactions in pubqueue")
	flags.BoolVar(&fnodeFlag.EnableRelay, "autorelay", false, "enable relay")
	flags.BoolVar(&fnodeFlag.EnableRelayService, "relayservice", false, "provide relay service")
}

func createPubQueueDb(path string) (*storage.QSBadger, error) {
	var err error
	pubQueueDb := storage.QSBadger{}
	err = pubQueueDb.Init(path + "_pubqueue")
	if err != nil {
		return nil, err
	}

	return &pubQueueDb, nil
}

func runFullnode(config cli.FullnodeFlag) {
	// NOTE: hardcode
	const defaultKeyName = "default"

	logger.Infof("Version: %s", utils.GitCommit)
	configLogger(fnodeFlag.IsDebug)

	signalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chain.SetAutoAck(config.AutoAck)

	peername := config.PeerName
	if config.IsBootstrap == true {
		peername = "bootstrap"
	}

	utils.EnsureDir(config.DataDir)

	//Load node options from config
	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	// overwrite by cli flags
	nodeoptions.IsRexTestMode = config.IsRexTestMode
	nodeoptions.EnableRelay = config.EnableRelay
	nodeoptions.EnableRelayService = config.EnableRelayService

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
	ds, err := dsbadger2.NewDatastore(path.Join(config.DataDir, fmt.Sprintf("%s-%s", peername, "peerstore")), &dsbadger2.DefaultOptions)
	CheckLockError(err)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	if config.IsBootstrap == true || config.EnableRelayService {
		//bootstrop/relay node connections: low watermarks: 1000  hi watermarks 50000, grace 30s
		cm, err := connmgr.NewConnManager(1000, 50000, connmgr.WithGracePeriod(30*time.Second))
		if err != nil {
			logger.Fatalf(err.Error())
		}
		node, err = p2p.NewNode(ctx, "", nodeoptions, config.IsBootstrap, ds, defaultkey, cm, config.ListenAddresses, config.JsonTracer)

		if err != nil {
			logger.Fatalf(err.Error())
		}

		datapath := config.DataDir + "/" + config.PeerName
		dbManager, err := storage.CreateDb(datapath)
		if err != nil {
			logger.Fatalf(err.Error())
		}
		dbManager.TryMigration(0) //TOFIX: pass the node data_ver
		dbManager.TryMigration(1)

		nodectx.InitCtx(ctx, "", node, dbManager, chainstorage.NewChainStorage(dbManager), "pubsub", utils.GitCommit)
		nodectx.GetNodeCtx().Keystore = ks
		nodectx.GetNodeCtx().PublicKey = keys.PubKey
		nodectx.GetNodeCtx().PeerId = peerid

		if err := stats.InitDB(datapath, node.Host.ID()); err != nil {
			logger.Fatalf("init stats db failed: %s", err)
		}

		logger.Infof("Host created, ID:<%s>, Address:<%s>", node.Host.ID(), node.Host.Addrs())
		h := &api.Handler{
			Node:      node,
			NodeCtx:   nodectx.GetNodeCtx(),
			GitCommit: utils.GitCommit,
		}
		startParam := api.StartAPIParam{
			IsDebug:       config.IsDebug,
			APIHost:       config.APIHost,
			APIPort:       config.APIPort,
			CertDir:       config.CertDir,
			ZeroAccessKey: config.ZeroAccessKey,
		}
		go api.StartAPIServer(startParam, signalch, h, nil, node, nodeoptions, ks, ethaddr, true)
	} else {
		nodename := "default"

		datapath := config.DataDir + "/" + config.PeerName
		dbManager, err := storage.CreateDb(datapath)
		if err != nil {
			logger.Fatalf(err.Error())
		}
		dbManager.TryMigration(0) //TOFIX: pass the node data_ver
		dbManager.TryMigration(1)
		newchainstorage := chainstorage.NewChainStorage(dbManager)

		//normal node connections: low watermarks: 10  hi watermarks 200, grace 60s
		cm, err := connmgr.NewConnManager(10, nodeoptions.ConnsHi, connmgr.WithGracePeriod(60*time.Second))
		if err != nil {
			logger.Fatalf(err.Error())
		}
		node, err = p2p.NewNode(ctx, nodename, nodeoptions, config.IsBootstrap, ds, defaultkey, cm, config.ListenAddresses, config.JsonTracer)
		if err == nil {
			node.SetRumExchange(ctx, newchainstorage)
		}

		if err := node.Bootstrap(ctx, config.BootstrapPeers); err != nil {
			logger.Fatal(err)
		}

		for _, addr := range node.Host.Addrs() {
			p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.Host.ID())
			logger.Infof("Peer ID:<%s>, Peer Address:<%s>", node.Host.ID(), p2paddr)
		}

		//Discovery and Advertise had been replaced by PeerExchange
		logger.Infof("Announcing ourselves...")
		discovery.Advertise(ctx, node.RoutingDiscovery, config.RendezvousString)
		logger.Infof("Successfully announced!")

		peerok := make(chan struct{})
		go node.ConnectPeers(ctx, peerok, nodeoptions.MaxPeers, config.RendezvousString)
		nodectx.InitCtx(ctx, nodename, node, dbManager, newchainstorage, "pubsub", utils.GitCommit)
		nodectx.GetNodeCtx().Keystore = ks
		nodectx.GetNodeCtx().PublicKey = keys.PubKey
		nodectx.GetNodeCtx().PeerId = peerid

		if err := stats.InitDB(datapath, node.Host.ID()); err != nil {
			logger.Fatalf("init stats db failed: %s", err)
		}

		//initial conn
		conn.InitConn()

		//initial group manager
		chain.InitGroupMgr()
		if nodeoptions.IsRexTestMode == true {
			chain.GetGroupMgr().SetRumExchangeTestMode()
		}

		// init the publish queue watcher
		doneCh := make(chan bool)
		pubqueueDb, err := createPubQueueDb(datapath)
		if err != nil {
			logger.Fatalf(err.Error())
		}
		chain.InitPublishQueueWatcher(doneCh, chain.GetGroupMgr(), pubqueueDb)

		//load all groups
		err = chain.GetGroupMgr().LoadAllGroups()
		if err != nil {
			logger.Fatalf(err.Error())
		}

		//start sync all groups
		err = chain.GetGroupMgr().StartSyncAllGroups()
		if err != nil {
			logger.Fatalf(err.Error())
		}

		appdb, err := appdata.CreateAppDb(datapath)
		if err != nil {
			logger.Fatalf(err.Error())
		}
		CheckLockError(err)

		//run local http api service
		h := &api.Handler{
			Node:       node,
			NodeCtx:    nodectx.GetNodeCtx(),
			Ctx:        ctx,
			GitCommit:  utils.GitCommit,
			Appdb:      appdb,
			ChainAPIdb: newchainstorage,
		}

		apiaddress := fmt.Sprintf("http://localhost:%d/api/v1", config.APIPort)
		appsync := appdata.NewAppSyncAgent(apiaddress, "default", appdb, dbManager)
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
		startParam := api.StartAPIParam{
			IsDebug:       config.IsDebug,
			APIHost:       config.APIHost,
			APIPort:       config.APIPort,
			CertDir:       config.CertDir,
			ZeroAccessKey: config.ZeroAccessKey,
		}
		go api.StartAPIServer(startParam, signalch, h, apph, node, nodeoptions, ks, ethaddr, false)
	}

	//attach signal
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	if config.IsBootstrap != true {
		//Stop sync all groups
		chain.GetGroupMgr().StopSyncAllGroups()
		//teardown all groups
		chain.GetGroupMgr().TeardownAllGroups()
		//close ctx db
		nodectx.GetDbMgr().CloseDb()
	}

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")
}

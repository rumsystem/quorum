package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	dsbadger2 "github.com/ipfs/go-ds-badger2"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	discovery "github.com/libp2p/go-libp2p-discovery"
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
	"github.com/spf13/cobra"
)

var (
	producerNodeFlag = cli.ProducerNodeFlag{ProtocolID: "/quorum/1.0.0"}
	producerNode     *p2p.Node
	producerSignalCh chan os.Signal
)

var producerNodeCmd = &cobra.Command{
	Use:   "producernode",
	Short: "Run producernode",
	Run: func(cmd *cobra.Command, args []string) {
		if producerNodeFlag.KeyStorePwd == "" {
			producerNodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runProducerNode(producerNodeFlag)
	},
}

func init() {
	rootCmd.AddCommand(producerNodeCmd)
	flags := producerNodeCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&producerNodeFlag.PeerName, "peername", "peer", "peername")
	flags.StringVar(&producerNodeFlag.ConfigDir, "configdir", "./config/", "config and keys dir")
	flags.StringVar(&producerNodeFlag.DataDir, "datadir", "./data/", "config dir")
	flags.StringVar(&producerNodeFlag.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flags.StringVar(&producerNodeFlag.KeyStoreName, "keystorename", "default", "keystore name")
	flags.StringVar(&producerNodeFlag.KeyStorePwd, "keystorepass", "", "keystore password")
	flags.Var(&producerNodeFlag.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.StringVar(&producerNodeFlag.APIHost, "apihost", "", "Domain or public ip addresses for api server")
	flags.UintVar(&producerNodeFlag.APIPort, "apiport", 5215, "api server listen port")
	flags.StringVar(&producerNodeFlag.CertDir, "certdir", "certs", "ssl certificate directory")
	flags.StringVar(&producerNodeFlag.ZeroAccessKey, "zerosslaccesskey", "", "zerossl access key, get from: https://app.zerossl.com/developer")
	flags.Var(&producerNodeFlag.BootstrapPeers, "peer", "bootstrap peer address")
	flags.StringVar(&producerNodeFlag.JsonTracer, "jsontracer", "", "output tracer data to a json file")
	flags.BoolVar(&producerNodeFlag.IsDebug, "debug", false, "show debug log")
}

func runProducerNode(config cli.ProducerNodeFlag) {
	configLogger(producerNodeFlag.IsDebug)
	logger.Infof("Version:%s", utils.GitCommit)
	const defaultKeyName = "default"

	producerSignalCh = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peername := config.PeerName

	utils.EnsureDir(config.DataDir)

	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	nodeoptions.IsRexTestMode = false
	nodeoptions.EnableRelay = false

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

	nodename := "producernode_default"

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
	producerNode, err = p2p.NewNode(ctx, nodename, nodeoptions, false, ds, defaultkey, cm, config.ListenAddresses, config.JsonTracer)
	if err == nil {
		producerNode.SetRumExchange(ctx, newchainstorage)
	}

	nodectx.InitCtx(ctx, nodename, producerNode, dbManager, newchainstorage, "pubsub", utils.GitCommit, nodectx.PRODUCER_NODE)
	nodectx.GetNodeCtx().Keystore = ks
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	nodectx.GetNodeCtx().PeerId = peerid

	if err := stats.InitDB(datapath, producerNode.Host.ID()); err != nil {
		logger.Fatalf("init stats db failed: %s", err)
	}

	//initial conn
	conn.InitConn()

	//initial group manager
	chain.InitGroupMgr()
	if nodeoptions.IsRexTestMode == true {
		chain.GetGroupMgr().SetRumExchangeTestMode()
	}

	//load all groups
	err = chain.GetGroupMgr().LoadAllGroups()
	if err != nil {
		logger.Fatalf(err.Error())
	}

	appdb, err := appdata.CreateAppDb(datapath)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	CheckLockError(err)

	if err := producerNode.Bootstrap(ctx, config.BootstrapPeers); err != nil {
		logger.Fatal(err)
	}

	for _, addr := range producerNode.Host.Addrs() {
		p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), producerNode.Host.ID())
		logger.Infof("Peer ID:<%s>, Peer Address:<%s>", producerNode.Host.ID(), p2paddr)
	}

	//Discovery and Advertise had been replaced by PeerExchange
	logger.Infof("Announcing ourselves...")
	discovery.Advertise(ctx, producerNode.RoutingDiscovery, config.RendezvousString)
	logger.Infof("Successfully announced!")

	peerok := make(chan struct{})
	go producerNode.ConnectPeers(ctx, peerok, nodeoptions.MaxPeers, config.RendezvousString)

	//start sync all groups
	err = chain.GetGroupMgr().StartSyncAllGroups()
	if err != nil {
		logger.Fatalf(err.Error())
	}

	//run local http api service
	h := &api.Handler{
		Node:       producerNode,
		NodeCtx:    nodectx.GetNodeCtx(),
		Ctx:        ctx,
		GitCommit:  utils.GitCommit,
		Appdb:      appdb,
		ChainAPIdb: newchainstorage,
	}

	startParam := api.StartServerParam{
		IsDebug:       config.IsDebug,
		APIHost:       config.APIHost,
		APIPort:       config.APIPort,
		CertDir:       config.CertDir,
		ZeroAccessKey: config.ZeroAccessKey,
	}

	go api.StartProducerServer(startParam, producerSignalCh, h, producerNode, nodeoptions, ks, ethaddr)

	//attach signal
	signal.Notify(producerSignalCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-producerSignalCh
	signal.Stop(producerSignalCh)

	//Stop sync all groups
	chain.GetGroupMgr().StopSyncAllGroups()
	//teardown all groups
	chain.GetGroupMgr().TeardownAllGroups()
	//close ctx db
	nodectx.GetDbMgr().CloseDb()

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")
}

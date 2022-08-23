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
	pNodeFlag = cli.ProducerNodeFlag{ProtocolID: "/quorum/1.0.0"}
	pNode     *p2p.Node
	pSignalCh chan os.Signal
)

var producerNodeCmd = &cobra.Command{
	Use:   "producernode",
	Short: " Run producernode",
	Run: func(cmd *cobra.Command, args []string) {
		if pNodeFlag.KeyStorePwd == "" {
			pNodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runProducerNode(pNodeFlag)
	},
}

func init() {
	rootCmd.AddCommand(producerNodeCmd)
	flags := producerNodeCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&pNodeFlag.PeerName, "peername", "peer", "peername")
	flags.StringVar(&pNodeFlag.ConfigDir, "configdir", "./config/", "config and keys dir")
	flags.StringVar(&pNodeFlag.DataDir, "datadir", "./data/", "config dir")
	flags.StringVar(&pNodeFlag.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flags.StringVar(&pNodeFlag.KeyStoreName, "keystorename", "default", "keystore name")
	flags.StringVar(&pNodeFlag.KeyStorePwd, "keystorepass", "", "keystore password")
	flags.Var(&pNodeFlag.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.StringVar(&pNodeFlag.APIHost, "apihost", "", "Domain or public ip addresses for api server")
	flags.UintVar(&pNodeFlag.APIPort, "apiport", 5215, "api server listen port")
	flags.StringVar(&pNodeFlag.CertDir, "certdir", "certs", "ssl certificate directory")
	flags.StringVar(&pNodeFlag.ZeroAccessKey, "zerosslaccesskey", "", "zerossl access key, get from: https://app.zerossl.com/developer")
	flags.Var(&pNodeFlag.BootstrapPeers, "peer", "bootstrap peer address")
	flags.StringVar(&pNodeFlag.JsonTracer, "jsontracer", "", "output tracer data to a json file")
	flags.BoolVar(&pNodeFlag.IsDebug, "debug", false, "show debug log")
}

func runProducerNode(config cli.ProducerNodeFlag) {
	configLogger(pNodeFlag.IsDebug)
	logger.Infof("Version:%s", utils.GitCommit)
	const defaultKeyName = "default"

	pSignalCh = make(chan os.Signal, 1)
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

	nodename := "pnode_default"

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
	pNode, err = p2p.NewNode(ctx, nodename, nodeoptions, false, ds, defaultkey, cm, config.ListenAddresses, config.JsonTracer)
	if err == nil {
		pNode.SetRumExchange(ctx, newchainstorage)
	}

	if err := pNode.Bootstrap(ctx, config.BootstrapPeers); err != nil {
		logger.Fatal(err)
	}

	for _, addr := range pNode.Host.Addrs() {
		p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), pNode.Host.ID())
		logger.Infof("Peer ID:<%s>, Peer Address:<%s>", pNode.Host.ID(), p2paddr)
	}

	//Discovery and Advertise had been replaced by PeerExchange
	logger.Infof("Announcing ourselves...")
	discovery.Advertise(ctx, pNode.RoutingDiscovery, config.RendezvousString)
	logger.Infof("Successfully announced!")

	peerok := make(chan struct{})
	go pNode.ConnectPeers(ctx, peerok, nodeoptions.MaxPeers, config.RendezvousString)
	nodectx.InitCtx(ctx, nodename, pNode, dbManager, newchainstorage, "pubsub", utils.GitCommit)
	nodectx.GetNodeCtx().Keystore = ks
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	nodectx.GetNodeCtx().PeerId = peerid

	if err := stats.InitDB(datapath, pNode.Host.ID()); err != nil {
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

	CheckLockError(err)

	//run local http api service
	h := &api.Handler{
		Node:       pNode,
		NodeCtx:    nodectx.GetNodeCtx(),
		Ctx:        ctx,
		GitCommit:  utils.GitCommit,
		Appdb:      nil,
		ChainAPIdb: newchainstorage,
	}

	startParam := api.StartProducerServerParam{
		IsDebug:       config.IsDebug,
		APIHost:       config.APIHost,
		APIPort:       config.APIPort,
		CertDir:       config.CertDir,
		ZeroAccessKey: config.ZeroAccessKey,
	}

	go api.StartProducerServer(startParam, pSignalCh, h, pNode, nodeoptions, ks, ethaddr)

	//attach signal
	signal.Notify(pSignalCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-pSignalCh
	signal.Stop(pSignalCh)

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

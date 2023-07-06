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
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/util"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
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
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	producerNodeFlag = cli.ProducerNodeFlag{ProtocolID: "/quorum/1.0.0"}
	producerNode     *p2p.Node
	producerViper    *viper.Viper
	producerSignalCh chan os.Signal
)

var producerNodeCmd = &cobra.Command{
	Use:   "producernode",
	Short: "Run producernode",
	Run: func(cmd *cobra.Command, args []string) {
		if err := producerViper.Unmarshal(&producerNodeFlag); err != nil {
			logger.Fatalf("viper unmarshal failed: %s", err)
		}

		if len(producerNodeFlag.ListenAddresses) == 0 {
			if len(producerViper.GetStringSlice("listen")) != 0 {
				addrlist, err := cli.ParseAddrList(strings.Join(producerViper.GetStringSlice("listen"), ","))
				if err != nil {
					logger.Fatalf("parse listen addr list failed: %s", err)
				}
				producerNodeFlag.ListenAddresses = *addrlist
			}
		}
		if len(producerNodeFlag.BootstrapPeers) == 0 {
			if len(producerViper.GetStringSlice("peer")) != 0 {
				addrlist, err := cli.ParseAddrList(strings.Join(producerViper.GetStringSlice("peer"), ","))
				if err != nil {
					logger.Fatalf("parse bootstrap peer addr list failed: %s", err)
				}
				producerNodeFlag.BootstrapPeers = *addrlist
			}
		}

		if producerNodeFlag.KeyStorePwd == "" {
			producerNodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runProducerNode(producerNodeFlag)
	},
}

func init() {
	producerViper = options.NewViper()

	rootCmd.AddCommand(producerNodeCmd)
	flags := producerNodeCmd.Flags()
	flags.SortFlags = false

	flags.String("peername", "peer", "peername")
	flags.String("configdir", "./config/", "config and keys dir")
	flags.String("datadir", "./data/", "config dir")
	flags.String("keystoredir", "./keystore/", "keystore dir")
	flags.String("keystorename", "default", "keystore name")
	flags.String("keystorepass", "", "keystore password")
	flags.StringSlice("listen", nil, "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.String("apihost", "localhost", "Domain or public ip addresses for api server")
	flags.Int("apiport", 5215, "api server listen port")
	flags.String("certdir", "certs", "ssl certificate directory")
	flags.String("zerosslaccesskey", "", "zerossl access key, get from: https://app.zerossl.com/developer")
	flags.StringSlice("peer", nil, "bootstrap peer address")
	flags.String("jsontracer", "", "output tracer data to a json file")
	flags.Bool("debug", false, "show debug log")

	if err := producerViper.BindPFlags(flags); err != nil {
		logger.Fatalf("viper bind flags failed: %s", err)
	}
}

func runProducerNode(config cli.ProducerNodeFlag) {
	color.Green("Version:%s", utils.GitCommit)
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
	producerNode, err = p2p.NewNode(ctx, nodename, nodeoptions, false, defaultkey, cm, config.ListenAddresses, []string{}, config.JsonTracer)
	if err == nil {
		producerNode.SetRumExchange(ctx)
	}

	nodectx.InitCtx(ctx, nodename, producerNode, dbManager, newchainstorage, "pubsub", utils.GitCommit, nodectx.PRODUCER_NODE)
	nodectx.GetNodeCtx().Keystore = ks
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	nodectx.GetNodeCtx().PeerId = peerid

	//initial conn
	conn.InitConn()

	//initial group manager
	chain.InitGroupMgr()

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

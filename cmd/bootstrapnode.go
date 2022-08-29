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
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/cli"
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
	bootstrapNodeFlag = cli.BootstrapNodeFlag{ProtocolID: "/quorum/1.0.0"}
	bootstrapNode     *p2p.Node
	bootstrapSignalch chan os.Signal
)

var bootstrapNodeCmd = &cobra.Command{
	Use:   "bootstrapnode",
	Short: "Run bootstrapnode",
	Run: func(cmd *cobra.Command, args []string) {
		if bootstrapNodeFlag.KeyStorePwd == "" {
			bootstrapNodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runBootstrapNode(bootstrapNodeFlag)
	},
}

func init() {
	rootCmd.AddCommand(bootstrapNodeCmd)

	flags := bootstrapNodeCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&bootstrapNodeFlag.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flags.StringVar(&bootstrapNodeFlag.KeyStoreName, "keystorename", "default", "keystore name")
	flags.StringVar(&bootstrapNodeFlag.KeyStorePwd, "keystorepass", "", "keystore password")
	flags.StringVar(&bootstrapNodeFlag.ConfigDir, "configdir", "./config/", "config and keys dir")
	flags.StringVar(&bootstrapNodeFlag.DataDir, "datadir", "./data/", "config dir")
	flags.Var(&bootstrapNodeFlag.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.BoolVar(&bootstrapNodeFlag.IsRexTestMode, "rextest", false, "RumExchange Test Mode")
	flags.BoolVar(&bootstrapNodeFlag.EnableRelay, "autorelay", true, "enable relay")
}

func runBootstrapNode(config cli.BootstrapNodeFlag) {
	// NOTE: hardcode
	const defaultKeyName = "default"

	logger.Infof("Version: %s", utils.GitCommit)
	configLogger(bootstrapNodeFlag.IsDebug)

	bootstrapSignalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//??
	chain.SetAutoAck(config.AutoAck)

	peername := "bootstrap"

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

	//bootstrop/relay node connections: low watermarks: 1000  hi watermarks 50000, grace 30s
	cm, err := connmgr.NewConnManager(1000, 50000, connmgr.WithGracePeriod(30*time.Second))
	if err != nil {
		logger.Fatalf(err.Error())
	}

	bootstrapNode, err = p2p.NewNode(ctx, "", nodeoptions, true, ds, defaultkey, cm, config.ListenAddresses, config.JsonTracer)

	if err != nil {
		logger.Fatalf(err.Error())
	}

	datapath := config.DataDir + "/" + config.PeerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf(err.Error())
	}
	//commented by cuicat
	//dbManager.TryMigration(0) //TOFIX: pass the node data_ver
	//dbManager.TryMigration(1)

	nodectx.InitCtx(ctx, "", bootstrapNode, dbManager, chainstorage.NewChainStorage(dbManager), "pubsub", utils.GitCommit, nodectx.BOOTSTRAP_NODE)
	nodectx.GetNodeCtx().Keystore = ks
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	nodectx.GetNodeCtx().PeerId = peerid

	if err := stats.InitDB(datapath, bootstrapNode.Host.ID()); err != nil {
		logger.Fatalf("init stats db failed: %s", err)
	}

	logger.Infof("usernode host created, ID:<%s>, Address:<%s>", bootstrapNode.Host.ID(), bootstrapNode.Host.Addrs())
	h := &api.Handler{
		Node:      bootstrapNode,
		NodeCtx:   nodectx.GetNodeCtx(),
		GitCommit: utils.GitCommit,
	}
	startParam := api.StartServerParam{
		IsDebug:       config.IsDebug,
		APIHost:       config.APIHost,
		APIPort:       config.APIPort,
		CertDir:       config.CertDir,
		ZeroAccessKey: config.ZeroAccessKey,
	}
	go api.StartBootstrapNodeServer(startParam, bootstrapSignalch, h, nil, bootstrapNode, nodeoptions, ks, ethaddr)

	//attach signal
	signal.Notify(bootstrapSignalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-bootstrapSignalch
	signal.Stop(bootstrapSignalch)

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")

}

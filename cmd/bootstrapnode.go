package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/api"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
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

	flags.StringVar(&bootstrapNodeFlag.APIHost, "apihost", "127.0.0.1", "Domain or public ip addresses for api server")
	flags.UintVar(&bootstrapNodeFlag.APIPort, "apiport", 4216, "api server listen port")
	flags.BoolVar(&bootstrapNodeFlag.EnableRelay, "autorelay", true, "enable relay")
}

func runBootstrapNode(config cli.BootstrapNodeFlag) {
	// NOTE: hardcode
	const defaultKeyName = "default"

	color.Green("Version: %s", utils.GitCommit)

	bootstrapSignalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peername := "bootstrap"

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

	//bootstrop/relay node connections: low watermarks: 1000  hi watermarks 50000, grace 30s
	cm, err := connmgr.NewConnManager(1000, 50000, connmgr.WithGracePeriod(30*time.Second))
	if err != nil {
		logger.Fatalf(err.Error())
	}

	bootstrapNode, err = p2p.NewNode(ctx, "", nodeoptions, true, defaultkey, cm, config.ListenAddresses, []string{}, config.JsonTracer)

	if err != nil {
		logger.Fatalf(err.Error())
	}

	datapath := config.DataDir + "/" + config.PeerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	nodectx.InitCtx(ctx, "", bootstrapNode, dbManager, chainstorage.NewChainStorage(dbManager), "pubsub", utils.GitCommit, nodectx.BOOTSTRAP_NODE)
	nodectx.GetNodeCtx().Keystore = ks
	nodectx.GetNodeCtx().PublicKey = keys.PubKey
	nodectx.GetNodeCtx().PeerId = peerid

	logger.Infof("bootstrap host created, ID:<%s>, Address:<%s>", bootstrapNode.Host.ID(), bootstrapNode.Host.Addrs())
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

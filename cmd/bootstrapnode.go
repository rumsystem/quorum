package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
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
	"github.com/spf13/viper"
)

var (
	bootstrapNodeFlag = cli.BootstrapNodeFlag{ProtocolID: "/quorum/1.0.0"}
	bootstrapNode     *p2p.Node
	bootstrapSignalch chan os.Signal
	bootstrapViper    *viper.Viper
)

var bootstrapNodeCmd = &cobra.Command{
	Use:   "bootstrapnode",
	Short: "Run bootstrapnode",
	Run: func(cmd *cobra.Command, args []string) {
		if err := bootstrapViper.Unmarshal(&bootstrapNodeFlag); err != nil {
			logger.Fatalf("viper unmarshal failed: %s", err)
		}

		if len(bootstrapNodeFlag.ListenAddresses) == 0 {
			if len(bootstrapViper.GetStringSlice("listen")) != 0 {
				addrlist, err := cli.ParseAddrList(strings.Join(bootstrapViper.GetStringSlice("listen"), ","))
				if err != nil {
					logger.Fatalf("parse listen addr list failed: %s", err)
				}
				bootstrapNodeFlag.ListenAddresses = *addrlist
			}
		}
		if len(bootstrapNodeFlag.BootstrapPeers) == 0 {
			if len(bootstrapViper.GetStringSlice("peer")) != 0 {
				addrlist, err := cli.ParseAddrList(strings.Join(bootstrapViper.GetStringSlice("peer"), ","))
				if err != nil {
					logger.Fatalf("parse bootstrap peer addr list failed: %s", err)
				}
				bootstrapNodeFlag.BootstrapPeers = *addrlist
			}
		}

		if bootstrapNodeFlag.KeyStorePwd == "" {
			bootstrapNodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runBootstrapNode(bootstrapNodeFlag)
	},
}

func init() {
	bootstrapViper = options.NewViper()

	rootCmd.AddCommand(bootstrapNodeCmd)

	flags := bootstrapNodeCmd.Flags()
	flags.SortFlags = false

	flags.String("keystoredir", "./keystore/", "keystore dir")
	flags.String("keystorename", "default", "keystore name")
	flags.String("keystorepass", "", "keystore password")
	flags.String("configdir", "./config/", "config and keys dir")
	flags.String("datadir", "./data/", "data dir")
	flags.StringSlice("listen", nil, "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")

	flags.String("apihost", "127.0.0.1", "Domain or public ip addresses for api server")
	flags.Int("apiport", 4216, "api server listen port")
	flags.Bool("autorelay", true, "enable relay")

	if err := bootstrapViper.BindPFlags(flags); err != nil {
		logger.Fatalf("viper bind flags failed: %s", err)
	}
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

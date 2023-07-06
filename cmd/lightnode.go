package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	nodesdkapi "github.com/rumsystem/quorum/pkg/nodesdk/api"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	lnodeFlag         = cli.LightnodeFlag{}
	lnodeViper        *viper.Viper
	lightnodeSignalch chan os.Signal
)

// lightnodeCmd represents the lightnode command
var lightnodeCmd = &cobra.Command{
	Use:   "lightnode",
	Short: "Run lightnode",
	Run: func(cmd *cobra.Command, args []string) {
		if err := lnodeViper.Unmarshal(&lnodeFlag); err != nil {
			logger.Fatalf("viper unmarshal failed: %s", err)
		}

		if lnodeFlag.KeyStorePwd == "" {
			lnodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		lnodeFlag.IsDebug = isDebug
		runLightnode(lnodeFlag)
	},
}

func init() {
	lnodeViper = options.NewViper()

	rootCmd.AddCommand(lightnodeCmd)

	flags := lightnodeCmd.Flags()
	flags.SortFlags = false

	flags.String("peername", "peer", "peername")
	flags.String("configdir", "./config/", "config and keys dir")
	flags.String("datadir", "./data/", "config dir")
	flags.String("keystoredir", "./keystore/", "keystore dir")
	flags.String("keystorename", "defaultkeystore", "keystore name")
	flags.String("keystorepass", "", "keystore password")
	flags.String("apihost", "", "Domain or public ip addresses for api server")
	flags.Int("apiport", 5215, "api server listen port")
	flags.String("jsontracer", "", "output tracer data to a json file")

	if err := lnodeViper.BindPFlags(flags); err != nil {
		logger.Fatalf("viper bind flags failed: %s", err)
	}
}

func runLightnode(config cli.LightnodeFlag) {
	color.Green("Version: %s", utils.GitCommit)
	const defaultKeyName = "nodesdk_default"

	lightnodeSignalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peername := config.PeerName

	if err := utils.EnsureDir(config.DataDir); err != nil {
		logger.Fatalf("check or create directory: %s failed: %s", config.DataDir, err)
	}

	//Load node options
	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	keystoreParam := InitKeystoreParam{
		KeystoreName:   config.KeyStoreName,
		KeystoreDir:    config.KeyStoreDir,
		KeystorePwd:    config.KeyStorePwd,
		DefaultKeyName: defaultKeyName,
		ConfigDir:      config.ConfigDir,
		PeerName:       config.PeerName,
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

	nodename := "nodesdk_default"

	datapath := config.DataDir + "/" + config.PeerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf(err.Error())
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

	//start node sdk server
	startApiParam := nodesdkapi.StartAPIParam{
		IsDebug: config.IsDebug,
		APIHost: config.APIHost,
		APIPort: config.APIPort,
	}
	go nodesdkapi.StartNodeSDKServer(startApiParam, lightnodeSignalch, nodeHandler, nodeoptions)

	//attach signal
	signal.Notify(lightnodeSignalch, os.Interrupt, syscall.SIGTERM)
	signalType := <-lightnodeSignalch
	signal.Stop(lightnodeSignalch)

	nodesdkctx.GetDbMgr().CloseDb()

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")
}

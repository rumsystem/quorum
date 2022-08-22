package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/autorelay"
	"github.com/rumsystem/quorum/pkg/autorelay/api"
	"github.com/spf13/cobra"
)

var rnodeFlag = cli.RelayNodeFlag{}
var relaynodeCmd = &cobra.Command{
	Use:   "relaynode",
	Short: "Run relaynode",
	Run: func(cmd *cobra.Command, args []string) {
		if rnodeFlag.KeyStorePwd == "" {
			rnodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runRelaynode(rnodeFlag)
	},
}

func init() {
	rootCmd.AddCommand(relaynodeCmd)

	flags := relaynodeCmd.Flags()
	flags.SortFlags = false

	flags.Var(&rnodeFlag.BootstrapPeers, "peer", "bootstrap peer address")
	flags.Var(&rnodeFlag.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.StringVar(&rnodeFlag.APIHost, "apihost", "", "Domain or public ip addresses for api server")
	flags.UintVar(&rnodeFlag.APIPort, "apiport", 5215, "api server listen port")
	flags.StringVar(&rnodeFlag.PeerName, "peername", "peer", "peername")
	flags.StringVar(&rnodeFlag.DataDir, "datadir", "./data/", "data dir")
	flags.StringVar(&rnodeFlag.ConfigDir, "configdir", "./config/", "config and keys dir")
	flags.StringVar(&rnodeFlag.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flags.StringVar(&rnodeFlag.KeyStoreName, "keystorename", "defaultkeystore", "keystore name")
	flags.StringVar(&rnodeFlag.KeyStorePwd, "keystorepass", "", "keystore password")
	flags.BoolVar(&rnodeFlag.IsDebug, "debug", false, "show debug log")

	relaynodeCmd.MarkFlagRequired("peer")
}

func runRelaynode(config cli.RelayNodeFlag) {
	// NOTE: hardcode
	const defaultKeyName = "relaynode_default"

	signalch := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Infof("Version: %s", utils.GitCommit)
	peername := config.PeerName

	relayNodeOpt, err := options.InitRelayNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	ks, defaultkey, err := InitRelayNodeKeystore(config, defaultKeyName, relayNodeOpt)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	_, err = localcrypto.SignKeytoPeerKeys(defaultkey)
	if err != nil {
		logger.Fatalf(err.Error())
		cancel()
	}

	peerid, ethaddr, err := ks.GetPeerInfo(defaultKeyName)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	logger.Infof("peer ID: <%s>", peerid)
	logger.Infof("eth addresss: <%s>", ethaddr)

	rdb, err := storage.InitRelayDb(config.DataDir + "/" + config.PeerName)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	relayNode, err := p2p.NewRelayServiceNode(ctx, relayNodeOpt, defaultkey, config.ListenAddresses, rdb)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	err = relayNode.Bootstrap(ctx, config.BootstrapPeers)
	if err != nil {
		cancel()
		logger.Fatalf(err.Error())
	}

	// now start relay api server
	handler := api.NewRelayServerHandler(rdb, relayNode)

	go autorelay.StartRelayServer(config, signalch, &handler)

	//attach signal
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")
}

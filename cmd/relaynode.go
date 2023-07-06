package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/autorelay"
	"github.com/rumsystem/quorum/pkg/autorelay/api"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rnodeFlag = cli.RelayNodeFlag{}
var rnodeViper *viper.Viper
var relaynodeCmd = &cobra.Command{
	Use:   "relaynode",
	Short: "Run relaynode",
	Run: func(cmd *cobra.Command, args []string) {
		if err := rnodeViper.Unmarshal(&rnodeFlag); err != nil {
			logger.Fatalf("viper unmarshal failed: %s", err)
		}

		if len(rnodeFlag.ListenAddresses) == 0 {
			if len(rnodeViper.GetStringSlice("listen")) != 0 {
				addrlist, err := cli.ParseAddrList(strings.Join(rnodeViper.GetStringSlice("listen"), ","))
				if err != nil {
					logger.Fatalf("parse listen addr list failed: %s", err)
				}
				rnodeFlag.ListenAddresses = *addrlist
			}
		}
		if len(rnodeFlag.BootstrapPeers) == 0 {
			if len(rnodeViper.GetStringSlice("peer")) != 0 {
				addrlist, err := cli.ParseAddrList(strings.Join(rnodeViper.GetStringSlice("peer"), ","))
				if err != nil {
					logger.Fatalf("parse bootstrap peer addr list failed: %s", err)
				}
				rnodeFlag.BootstrapPeers = *addrlist
			}
		}

		if rnodeFlag.KeyStorePwd == "" {
			rnodeFlag.KeyStorePwd = os.Getenv("RUM_KSPASSWD")
		}
		runRelaynode(rnodeFlag)
	},
}

func init() {
	rnodeViper = options.NewViper()

	rootCmd.AddCommand(relaynodeCmd)

	flags := relaynodeCmd.Flags()
	flags.SortFlags = false

	flags.StringSlice("peer", nil, "bootstrap peer address")
	flags.StringSlice("listen", nil, "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	flags.String("apihost", "", "Domain or public ip addresses for api server")
	flags.Int("apiport", 5215, "api server listen port")
	flags.String("peername", "peer", "peername")
	flags.String("datadir", "./data/", "data dir")
	flags.String("configdir", "./config/", "config and keys dir")
	flags.String("keystoredir", "./keystore/", "keystore dir")
	flags.String("keystorename", "defaultkeystore", "keystore name")
	flags.String("keystorepass", "", "keystore password")
	flags.Bool("debug", false, "show debug log")

	if err := rnodeViper.BindPFlags(flags); err != nil {
		logger.Fatalf("viper bind flags failed: %s", err)
	}
}

func runRelaynode(config cli.RelayNodeFlag) {
	// NOTE: hardcode
	const defaultKeyName = "relaynode_default"

	signalch := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	color.Green("Version: %s", utils.GitCommit)
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

	bucket := "relaydb"
	rdb, err := storage.NewStore(ctx, config.DataDir+"/"+config.PeerName, bucket)
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
	signal.Notify(signalch, os.Interrupt, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	//cleanup before exit
	logger.Infof("On Signal <%s>", signalType)
	logger.Infof("Exit command received. Exiting...")
}

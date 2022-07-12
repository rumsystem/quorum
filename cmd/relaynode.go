/* for auto relay service node */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

const DEFAUT_KEY_NAME string = "relaynode_default"

var (
	ReleaseVersion string
	GitCommit      string
	signalch       chan os.Signal
	mainlog        = logging.Logger("relaynode")
)

func main() {
	if ReleaseVersion == "" {
		ReleaseVersion = "v1.0.0"
	}

	if GitCommit == "" {
		GitCommit = "devel"
	}

	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")

	config, err := cli.ParseRelayNodeFlags()

	lvl, err := logging.LevelFromString("info")
	logging.SetAllLoggers(lvl)

	if err != nil {
		panic(err)
	}

	if config.IsDebug == true {
		logging.SetLogLevel("nodesdk", "debug")
	}

	if *help {
		fmt.Println("Output a help ")
		fmt.Println()
		fmt.Println("Usage:...")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Printf("%s - %s\n", ReleaseVersion, GitCommit)
		return
	}

	os.Exit(runRelayNodeRet(config))
}

func runRelayNodeRet(config cli.RelayNodeConfig) int {
	signalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mainlog.Infof("Version: %s", GitCommit)
	peername := config.PeerName

	relayNodeOpt, err := options.InitRelayNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	ks, defaultkey, err := InitRelayNodeKeystore(config, relayNodeOpt)

	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	_, err = localcrypto.SignKeytoPeerKeys(defaultkey)

	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}

	peerid, ethaddr, err := ks.GetPeerInfo(DEFAUT_KEY_NAME)

	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	mainlog.Infof("peer ID: <%s>", peerid)
	mainlog.Infof("eth addresss: <%s>", ethaddr)

	relayNode, err := p2p.NewRelayServiceNode(ctx, relayNodeOpt, defaultkey, config.ListenAddresses)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	err = relayNode.Bootstrap(ctx, config)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	//attach signal
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	//cleanup before exit
	mainlog.Infof("On Signal <%s>", signalType)
	mainlog.Infof("Exit command received. Exiting...")

	return 0
}

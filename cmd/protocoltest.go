package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

var mainlog = logging.Logger("main")

func main() {
	logging.SetLogLevel("main", "debug")
	config, err := cli.ParseFlags()
	if err != nil {
		panic(err)
	}
	bootpeers := config.BootstrapPeers
	listenaddresses, _ := utils.StringsToAddrs([]string{config.ListenAddresses})

	bootpeer, err := peer.AddrInfoFromP2pAddr(bootpeers[0])
	if err != nil {
		panic(err)
	}
	fmt.Printf("connect to %s\n", bootpeer)

	keys, _ := localcrypto.LoadKeys(config.ConfigDir, "testprotocol")
	peerid, err := peer.IDFromPublicKey(keys.PubKey)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	var ddht *dual.DHT
	var routingDiscovery *discovery.RoutingDiscovery
	routing := libp2p.Routing(func(host host.Host) (routing.PeerRouting, error) {
		dhtOpts := dual.DHTOption(
			dht.Mode(dht.ModeServer),
			dht.Concurrency(10),
		)

		var err error
		ddht, err = dual.New(ctx, host, dhtOpts)
		routingDiscovery = discovery.NewRoutingDiscovery(ddht)
		return ddht, err
	})

	fmt.Println(bootpeer)
	fmt.Println(peerid)

	identity := libp2p.Identity(keys.PrivKey)

	host, err := libp2p.New(ctx,
		routing,
		libp2p.ListenAddrs(listenaddresses...),
		libp2p.ConnectionManager(connmgr.NewConnManager(10, 200, 60)),
		identity,
	)
	if err != nil {
		panic(err)
	}

	if bootpeer.ID != host.ID() {
		pctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		err := host.Connect(pctx, *bootpeer)
		if err != nil {
			mainlog.Warningf("connect peer failure: %s \n", bootpeer)
			cancel()
		}
	}

	protocols := []string{"/ipfs/id/1.0.0", "/ipfs/kad/1.0.0", "/ipfs/ping/1.0.0", "/randomsub/1.0.0", "/floodsub/1.0.0", "/meshsub/1.1.0", "/meshsub/1.0.0", "/quorum/nevis/kad/1.0.0", "/quorum/nevis/meshsub/1.1.0", "/quorum/nevis/meshsub/1.0.0", "/quorum/nevis/floodsub/1.0.0"}

	for _, protocolid := range protocols {
		_, err = host.NewStream(ctx, bootpeer.ID, protocol.ID(protocolid))
		if err != nil {
			fmt.Printf("Stream open failed %s %s\n", err, protocolid)
		} else {
			fmt.Printf("protocol exist: %s\n", protocolid)

		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-ch
	signal.Stop(ch)
	mainlog.Infof("On Signal <%s>", signalType)
	mainlog.Infof("Exit command received. Exiting...")
}

package cmd

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping peer",
	Run: func(cmd *cobra.Command, args []string) {
		ping(peerList)
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	flags := pingCmd.Flags()
	flags.SortFlags = false
	flags.VarP(&peerList, "peer", "p", "peer address")
	pingCmd.MarkFlagRequired("peer")
}

func ping(peerList cli.AddrList) {
	tcpAddr := "/ip4/127.0.0.1/tcp/0"
	wsAddr := "/ip4/127.0.0.1/tcp/0/ws"
	ctx := context.Background()
	node, err := libp2p.New(
		libp2p.ListenAddrStrings(tcpAddr, wsAddr),
		libp2p.Ping(false),
	)
	if err != nil {
		logger.Fatal(err)
	}

	// configure our ping protocol
	pingService := &p2p.PingService{Host: node}
	node.SetStreamHandler(p2p.PingID, pingService.PingHandler)

	for _, addr := range peerList {
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			logger.Fatal(err)
		}

		if err := node.Connect(ctx, *peer); err != nil {
			logger.Fatal(err)
		}
		ch := pingService.Ping(ctx, peer.ID)
		fmt.Println()
		fmt.Println("pinging remote peer at", addr)
		for i := 0; i < 4; i++ {
			res := <-ch
			fmt.Println("PING", addr, "in", res.RTT)
		}
	}
}

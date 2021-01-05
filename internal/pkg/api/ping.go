package api

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/huo-ju/quorum/internal/pkg/p2p"
	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

type AddrProtoPair struct {
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Protocol string `json:"protocol"`
}

var networkLog = logging.Logger("network")

func (h *Handler) PingPeer(node *p2p.Node) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		output := make(map[string]interface{})

		pingService := ping.NewPingService(node.Host)
		pctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wg := new(sync.WaitGroup)
		for _, pid := range node.Host.Peerstore().Peers() {
			if node.Host.Network().Connectedness(pid) == network.Connected {
				/* copy peerId here to avoid pointer overwrite in goroutine */
				peerId := pid.String()
				pInfo := make(map[string]interface{})
				output[peerId] = pInfo

				ts := pingService.Ping(pctx, pid)
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case res := <-ts:
						if res.Error != nil {
							pInfo["rtt"] = res.Error.Error()
						} else {
							networkLog.Infof("%s: %s", peerId, res.RTT.String())
							pInfo["rtt"] = res.RTT.String()
						}
					case <-time.After(time.Second * 4):
						networkLog.Infof("%s: timedout", peerId)
					}
				}()

				addrs := []string{}
				for _, addr := range node.Host.Peerstore().Addrs(pid) {
					addrs = append(addrs, addr.String())
				}
				sort.Strings(addrs)
				pInfo["addrs"] = addrs

				protocols := []string{}
				pairs := []AddrProtoPair{}
				for _, c := range node.Host.Network().ConnsToPeer(pid) {
					for _, s := range c.GetStreams() {
						pairs = append(pairs, AddrProtoPair{c.LocalMultiaddr().String(), c.RemoteMultiaddr().String(), string(s.Protocol())})
						protocols = append(protocols, string(s.Protocol()))
					}
				}
				sort.Strings(protocols)
				pInfo["protocols"] = protocols
				pInfo["connections"] = pairs

			}
		}
		wg.Wait()

		return c.JSON(http.StatusOK, output)
	}
}

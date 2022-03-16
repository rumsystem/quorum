package api

import (
	"net/http"
	"sort"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

type AddrProtoPair struct {
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Protocol string `json:"protocol"`
}

var networkLog = logging.Logger("network")

// @Tags Node
// @Summary PingPeer
// @Description PingPeer
// @Accept json
// @Produce json
// @Success 200 {object} AddrProtoPair
// @Router /api/v1/network/peers/ping [get]
func (h *Handler) PingPeer(node *p2p.Node) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		output := make(map[string]interface{})

		wg := new(sync.WaitGroup)
		for _, pid := range node.Host.Peerstore().Peers() {
			if node.Host.Network().Connectedness(pid) == network.Connected {
				/* copy peerId here to avoid pointer overwrite in goroutine */
				peerId := pid.String()
				pInfo := make(map[string]interface{})
				output[peerId] = pInfo
				wg.Add(1)

				go func() {
					defer wg.Done()
					res, err := handlers.Ping(node.Pubsub, node.Host.ID(), peerId)
					if err != nil {
						pInfo["rtt"] = err.Error()
					} else {
						pInfo["rtt"] = res.TTL
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

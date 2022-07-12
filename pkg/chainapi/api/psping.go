package api

import (
	"net/http"
	"sort"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type PSPingParam struct {
	PeerId string `from:"peer_id"      json:"peer_id"      validate:"required,max=53,min=53"`
}

// @Tags Node
// @Summary PubsubPing
// @Description Pubsub ping utility
// @Accept json
// @Produce json
// @Param data body PSPingParam true "pingparam"
// @Success 200 {object} handlers.PingResp
// @Router /api/v1/psping [post]
func (h *Handler) PSPingPeer(node *p2p.Node) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		cc := c.(*utils.CustomContext)
		params := new(PSPingParam)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		result, err := handlers.Ping(node.Pubsub, node.Host.ID(), params.PeerId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

type AddrProtoPair struct {
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Protocol string `json:"protocol"`
}

// @Tags Node
// @Summary PingPeers
// @Description PingPeers will ping all peers via psping
// @Accept json
// @Produce json
// @Success 200 {object} AddrProtoPair
// @Router /api/v1/network/peers/ping [get]
func (h *Handler) PingPeers(node *p2p.Node) echo.HandlerFunc {
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

package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

// @Tags Node
// @Summary GetNetwork
// @Description Get node's network information
// @Produce json
// @Success 200 {object} handlers.NetworkInfo
// @Router /api/v1/network [get]
func (h *Handler) GetNetwork(nodehost *host.Host, nodeinfo *p2p.NodeInfo, nodeopt *options.NodeOptions, ethaddr string) echo.HandlerFunc {
	return func(c echo.Context) error {
		result, err := handlers.GetNetwork(nodehost, nodeinfo, nodeopt, ethaddr)
		if err != nil {
			fmt.Printf("json.Marshal failed: %s", err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

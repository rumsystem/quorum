package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
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

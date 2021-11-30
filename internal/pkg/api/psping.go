package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
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

		validate := validator.New()
		params := new(PSPingParam)
		output := make(map[string]interface{})

		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		result, err := handlers.Ping(node.Pubsub, node.Host.ID(), params.PeerId)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, result)
	}
}

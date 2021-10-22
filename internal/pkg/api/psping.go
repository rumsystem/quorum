package api

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	"net/http"
)

type PSPingParam struct {
	PeerId string `from:"peer_id"      json:"peer_id"      validate:"required,max=53,min=53"`
}

type PingResult struct {
	Result [10]int64 `json:"pingresult"`
}

// @Tags Node
// @Summary PubsubPing
// @Description Pubsub ping utility
// @Accept json
// @Produce json
// @Param data body PSPingParam true "pingparam"
// @Success 200 {object} PingResult
// @Router api/v1/psping [post]
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

		ctx, cancel := context.WithCancel(context.Background())
		psping := p2p.NewPSPingService(ctx, node.Pubsub, node.Host.ID())
		result, err := psping.PingReq(params.PeerId)
		defer cancel()

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		} else {

			return c.JSON(http.StatusOK, &PingResult{result})
		}
	}
}

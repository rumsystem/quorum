package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type P2PPingParam struct {
	PeerId string `from:"peer_id"      json:"peer_id"      validate:"required,max=53,min=53"`
}

// @Tags Node
// @Summary P2PPingPeer
// @Description P2PPingPeer is the same with ipfs ping, just rename the protocol name
// @Accept json
// @Produce json
// @Param data body P2PPingParam true "pingparam"
// @Success 200 {object} handlers.PingResp
// @Router /api/v1/psping [post]
func (h *Handler) P2PPingPeer(node *p2p.Node) echo.HandlerFunc {
	return func(c echo.Context) (err error) {

		validate := validator.New()
		params := new(P2PPingParam)
		output := make(map[string]interface{})

		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		result, err := handlers.P2PPing(node.Host, params.PeerId)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, result)
	}
}

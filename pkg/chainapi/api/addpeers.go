package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Node
// @Summary AddPeers
// @Description Connect to peers
// @Accept json
// @Produce json
// @Param data body []string true "Peers List"
// @Success 200 {object} handlers.AddPeerResult
// @Router /api/v1/network/peers [post]
func (h *Handler) AddPeers(c echo.Context) (err error) {
	params := new(handlers.AddPeerParam)
	if err := c.Bind(params); err != nil {
		rumerrors.NewBadRequestError(err.Error())
	}

	result, err := handlers.AddPeers(*params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, result)
}

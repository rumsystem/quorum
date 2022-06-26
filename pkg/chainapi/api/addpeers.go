package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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
	cc := c.(*utils.CustomContext)
	params := new(handlers.AddPeerParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	result, err := handlers.AddPeers(*params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, result)
}

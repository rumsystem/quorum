package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary GetGroupProducers
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.ProducerListItem
// @Router /api/v1/group/{group_id}/producers [get]
func (h *Handler) GetGroupProducers(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	res, err := handlers.GetGroupProducers(h.ChainAPIdb, groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

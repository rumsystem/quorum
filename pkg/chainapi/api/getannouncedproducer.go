package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags User
// @Summary GetAnnouncedGroupProducer
// @Description Get the list of announced group producers
// @Accept json
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} handlers.AnnouncedProducers
// @Router /api/v1/group/{group_id}/announced/producers [get]
func (h *Handler) GetAnnouncedProducers(c echo.Context) (err error) {
	groupid := c.Param("group_id")

	res, err := handlers.GetAnnouncedProducers(h.ChainAPIdb, groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

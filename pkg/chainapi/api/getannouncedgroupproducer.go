package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags User
// @Summary GetAnnouncedGroupProducer
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} handlers.AnnouncedProducerListItem
// @Router /api/v1/group/{group_id}/announced/producers [get]
func (h *Handler) GetAnnouncedGroupProducer(c echo.Context) (err error) {
	groupid := c.Param("group_id")

	res, err := handlers.GetAnnouncedGroupProducer(h.ChainAPIdb, groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

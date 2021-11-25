package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags User
// @Summary GetAnnouncedGroupProducer
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} handlers.AnnouncedProducerListItem
// @Router /api/v1/group/{group_id}/announced/producers [get]
func (h *Handler) GetAnnouncedGroupProducer(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")

	res, err := handlers.GetAnnouncedGroupProducer(groupid)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary GetGroupProducers
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.ProducerListItem
// @Router /api/v1/group/{group_id}/producers [get]
func (h *Handler) GetGroupProducers(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")

	res, err := handlers.GetGroupProducers(groupid)

	if groupid == "" {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary GetAppConfigItem
// @Description get app config item
// @Produce json
// @Param group_id path string true "Group Id"
// @Param key path string true "itemKey"
// @Success 200 {object} handlers.AppConfigKeyItem
// @Router /api/v1/group/{group_id}/config/{key} [get]
func (h *Handler) GetAppConfigItem(c echo.Context) (err error) {
	output := make(map[string]string)

	groupId := c.Param("group_id")
	itemKey := c.Param("key")

	res, err := handlers.GetAppConfigKey(itemKey, groupId)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

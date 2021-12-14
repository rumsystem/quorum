package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func (h *Handler) GetGroupConfigItem(c echo.Context) (err error) {
	output := make(map[string]string)

	groupId := c.Param("group_id")
	itemKey := c.Param("key")

	res, err := handlers.GetGroupConfigKey(itemKey, groupId)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

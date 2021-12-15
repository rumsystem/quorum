package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func (h *Handler) GetGroupConfigKey(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	res, err := handlers.GetGroupConfigKeyList(groupid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary MgrAppConfig
// @Description set app config
// @Produce json
// @Param data body handlers.AppConfigParam true "AppConfigParam"
// @Success 200 {object} handlers.AppConfigResult
// @Router /api/v1/group/appconfig [post]
func (h *Handler) MgrAppConfig(c echo.Context) (err error) {
	output := make(map[string]string)
	params := new(handlers.AppConfigParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.MgrAppConfig(params)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

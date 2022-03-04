package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary chainconfig
// @Description chain config
// @Accept json
// @Produce json
// @Param data body handlers.ChainConfigParams true "ChainConfigParams"
// @Success 200 {object} handlers.ChainConfigResult
// @Router /api/v1/group/chainconfig [post]
func (h *Handler) MgrChainConfig(c echo.Context) (err error) {
	output := make(map[string]string)
	params := new(handlers.ChainConfigParams)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.MgrChainConfig(params)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

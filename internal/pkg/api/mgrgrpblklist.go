package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary DeniedList
// @Description add or remove a user from the denied list
// @Accept json
// @Produce json
// @Param data body handlers.DenyListParam true "DenyListParam"
// @Success 200 {object} handlers.DenyUserResult
// @Router /api/v1/group/deniedlist [post]
func (h *Handler) MgrGrpBlkList(c echo.Context) (err error) {
	output := make(map[string]string)
	params := new(handlers.DenyListParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.MgrGrpBlkList(params)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

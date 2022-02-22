package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Groups
// @Summary ClearGroupData
// @Description Clear group data
// @Produce json
// @Success 200 {object} handlers.ClearGroupDataResult
// @Router /api/v1/group/clear [post]
func (h *Handler) ClearGroupData(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(handlers.ClearGroupDataParam)

	if err := c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.ClearGroupData(params)

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary ClearGroupData
// @Description Clear group data
// @Produce json
// @Success 200 {object} handlers.ClearGroupDataResult
// @Router /api/v1/group/clear [post]
func (h *Handler) ClearGroupData(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.ClearGroupDataParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.ClearGroupData(params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

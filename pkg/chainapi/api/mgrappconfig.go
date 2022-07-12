package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary MgrAppConfig
// @Description set app config
// @Produce json
// @Param data body handlers.AppConfigParam true "AppConfigParam"
// @Success 200 {object} handlers.AppConfigResult
// @Router /api/v1/group/appconfig [post]
func (h *Handler) MgrAppConfig(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	params := new(handlers.AppConfigParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.MgrAppConfig(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
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
	cc := c.(*utils.CustomContext)

	params := new(handlers.ChainConfigParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.MgrChainConfig(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

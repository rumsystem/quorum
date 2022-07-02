package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary GetChainTrxAuthMode
// @Description GetChainTrxAuthMode
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param trx_type path string  true "trxType"
// @Success 200 {array} handlers.TrxAuthItem
// @Router /api/v1/group/{group_id}/trx/auth/{trx_type} [get]
func (h *Handler) GetChainTrxAuthMode(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.TrxAuthParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.GetChainTrxAuthMode(h.ChainAPIdb, params.GroupId, params.TrxType)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

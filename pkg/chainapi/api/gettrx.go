package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Chain
// @Summary GetTrx
// @Description Get a transaction a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param trx_id path string  true "Transaction Id"
// @Success 200 {object} pb.Trx
// @Router /api/v1/trx/{group_id}/{trx_id} [get]
func (h *Handler) GetTrx(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	var params handlers.GetTrxParam
	if err := cc.BindAndValidate(&params); err != nil {
		return err
	}

	trx, err := handlers.GetTrx(params.GroupId, params.TrxId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, trx)
}

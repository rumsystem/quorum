package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
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
	groupid := c.Param("group_id")
	trxType := c.Param("trx_type")
	res, err := handlers.GetChainTrxAuthMode(h.ChainAPIdb, groupid, trxType)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	return c.JSON(http.StatusOK, res)
}

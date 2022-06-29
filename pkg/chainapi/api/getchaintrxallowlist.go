package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary GetChainTrxAllowList
// @Description Get group allow list
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.ChainSendTrxRuleListItem
// @Router /api/v1/group/{group_id}/trx/allowlist [get]
func (h *Handler) GetChainTrxAllowList(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	res, err := handlers.GetChainTrxAllowList(h.ChainAPIdb, groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary GetChainTrxAllowList
// @Description Get group allow list
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.ChainSendTrxRuleListItem
// @Router /api/v1/group/{group_id}/trx/allowlist [get]
func (h *Handler) GetChainTrxAllowList(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	res, err := handlers.GetChainTrxAllowList(groupid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}

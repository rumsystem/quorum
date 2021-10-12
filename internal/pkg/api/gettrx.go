package api

import (
	"fmt"
	"net/http"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	_ "github.com/rumsystem/quorum/internal/pkg/pb" //import for swaggo
	"github.com/labstack/echo/v4"
)

type GetTrxParam struct {
	TrxId string `from:"trx_id" json:"trx_id" validate:"required"`
}

// @Tags Chain
// @Summary GetTrx
// @Description Get a transaction a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param trx_id path string  true "Transaction Id"
// @Success 200 {object} pb.Trx
// @Router /api/v1/trx/{group_id}/{trx_id} [get]
func (h *Handler) GetTrx(c echo.Context) (err error) {

	output := make(map[string]string)

	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	trxid := c.Param("trx_id")
	if trxid == "" {
		output[ERROR_INFO] = "trx_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		trx, err := group.GetTrx(trxid)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, trx)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

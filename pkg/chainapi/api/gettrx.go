package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/pb"
)

type GetTrxResponse struct {
	Trx    *pb.Trx `json:"trx"`
	Status string  `json:"status"`
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
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	trxid := c.Param("trx_id")
	if trxid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidTrxID)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		trx, isOnChain, err := group.GetTrx(trxid)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		resp := &GetTrxResponse{
			Trx: trx,
		}

		if isOnChain {
			resp.Status = "onchain"
		} else {
			resp.Status = "offchain"
		}
		return c.JSON(http.StatusOK, resp)
	} else {
		return rumerrors.NewBadRequestError(fmt.Sprintf("Group %s not exist", groupid))
	}
}

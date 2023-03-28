package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

// @Tags NodeAPI
// @Summary GetContentNSdk
// @Description get content
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param   get_content_params  query handlers.GetGroupCtnPrarms  true  "get group content params"
// @Success 200 {object} []quorumpb.Trx
// @Router  /api/v1/node/{group_id}/groupctn [get]
func (h *Handler) GetContentNSdk(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.GetGroupCtnPrarms)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	}

	if params.Num == 0 {
		params.Num = 20
	}

	trxids, err := h.Appdb.GetGroupContentBySenders(
		params.GroupId,
		params.Senders,
		params.StartTrx,
		params.Num,
		params.Reverse,
		params.IncludeStartTrx,
	)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	trxList := []*quorumpb.Trx{}
	for _, trxid := range trxids {
		trx, err := group.GetTrx(trxid)
		if err != nil {
			c.Logger().Errorf("GetTrx Err: %s", err)
			continue
		}
		if trx != nil && trx.TrxId != "" {
			trxList = append(trxList, trx)
		}
	}

	return c.JSON(http.StatusOK, trxList)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type SendTrxResult struct {
	TrxId string `json:"trx_id"   validate:"required,uuid4"`
}

type NodeSDKTrxItem struct {
	TrxBytes []byte
}

type NSdkSendTrxParams struct {
	GroupId string        `json:"-" param:"group_id" validate:"required" example:"630a545b-1ff4-4b9e-9a5d-bb13b6f6a629"`
	Trx     *quorumpb.Trx `json:"trx" validate:"required"`
}

// @Tags LightNode
// @Summary NSdkSendTrx
// @Description send trx
// @Produce json
// @Accept  json
// @Param   group_id path string true "Group Id"
// @Param   data body NSdkSendTrxParams true "send trx params"
// @Success 200 {object} SendTrxResult
// @Router /api/v1/node/{group_id}/trx [post]
func (h *Handler) NSdkSendTrx(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	payload := new(NSdkSendTrxParams)
	if err := cc.BindAndValidate(payload); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[payload.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	}

	//check if trx sender is in group block list
	trx := payload.Trx
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, nodectx.GetNodeCtx().Name)
	if err != nil {
		return rumerrors.NewUnauthorizedError(err)
	}

	if !isAllow {
		return rumerrors.NewForbiddenError()
	}

	trxId, err := group.SendRawTrx(trx)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	sendTrxResult := &SendTrxResult{TrxId: trxId}
	return c.JSON(http.StatusOK, sendTrxResult)
}

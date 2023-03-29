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

type NSdkSendTrxParams struct {
	GroupId      string `json:"-" param:"group_id" validate:"required" example:"630a545b-1ff4-4b9e-9a5d-bb13b6f6a629"`
	TrxId        string `json:"trx_id" validate:"required" example:"9428e2f7-b585-4c0b-bc8f-835482cb2f26"`
	Data         []byte `json:"data" validate:"required" swaggertype:"string" format:"base64" example:"eyJ0eXBlIjogIkxpa2UiLCAib2JqZWN0IjogeyJ0eXBlIjogIk5vdGUiLCAiaWQiOiAiN2I2NWViYzgtZTk0ZS00ZjBhLWIxNGUtOTMwYWYyOWQyNDgxIn19"`
	Version      string `json:"version" validate:"required" example:"2.0.0"`
	SenderPubkey string `json:"sender_pubkey" validate:"required" example:"A0rTbiviB1KMgrcehCU9uYiEM2oSUigv_qdCJgW_etO3"`
	SenderSign   []byte `json:"sender_sign" validate:"required" swaggertype:"string" format:"base64" example:"d3DGzZbDZr/exsbHsUK0VRbreIBjKP4bON0+N2Ri3BZdHOn2znf1LXu4QYSkZlr5RVNE946G92gKNlRfuF0/IwE="`
	TimeStamp    int64  `json:"timestamp" example:"1680066699716"` // millisecond
	Expired      int64  `json:"expired" example:"1680067046530"`
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

	trx := &quorumpb.Trx{
		TrxId:        payload.TrxId,
		GroupId:      payload.GroupId,
		Data:         payload.Data,
		TimeStamp:    payload.TimeStamp,
		Version:      payload.Version,
		Expired:      payload.Expired,
		SenderPubkey: payload.SenderPubkey,
		SenderSign:   payload.SenderSign,
	}
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

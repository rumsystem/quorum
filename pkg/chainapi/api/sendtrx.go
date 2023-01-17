package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type SendTrxResult struct {
	TrxId string `json:"trx_id"   validate:"required"`
}

type NodeSDKTrxItem struct {
	TrxBytes []byte
}

type NodeSDKSendTrxItem struct {
	GroupId string `param:"group_id" json:"-" validate:"required" example:"630a545b-1ff4-4b9e-9a5d-bb13b6f6a629"`
	TrxItem []byte `json:"TrxItem" validate:"required" swaggertype:"primitive,string"` // base64 encoded trx data
}

// discuss with wanming
func is_user_blocked(c echo.Context) bool {
	return false
}

// @Tags LightNode
// @Summary SendTrx
// @Description send trx
// @Produce json
// @Accept  json
// @Param   group_id path string true "Group Id"
// @Param   send_trx_params  body NodeSDKSendTrxItem true  "send trx params"
// @Success 200 {object} SendTrxResult
// @Router /api/v1/node/trx/{group_id} [post]
func (h *Handler) SendTrx(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	if is_user_blocked(c) {
		return rumerrors.NewBadRequestError("BLOCKED_USER")
	}

	sendTrxItem := new(NodeSDKSendTrxItem)
	if err := cc.BindAndValidate(sendTrxItem); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[sendTrxItem.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	}

	ciperKey, err := hex.DecodeString(group.Item.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	decryptData, err := localcrypto.AesDecode(sendTrxItem.TrxItem, ciperKey)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	trxItem := new(NodeSDKTrxItem)

	if err := json.Unmarshal(decryptData, trxItem); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	trx := new(quorumpb.Trx)
	err = proto.Unmarshal(trxItem.TrxBytes, trx)
	if err != nil {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidTrxData)
	}

	//check if trx sender is in group block list
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

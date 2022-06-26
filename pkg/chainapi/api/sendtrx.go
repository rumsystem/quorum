package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type SendTrxResult struct {
	TrxId string `json:"trx_id"   validate:"required"`
}

type NodeSDKTrxItem struct {
	TrxBytes []byte
	JwtToken string
}

type NodeSDKSendTrxItem struct {
	GroupId string `param:"group_id" validate:"required"`
	TrxItem []byte
}

//discuss with wanming
func is_user_blocked(c echo.Context) bool {
	return false
}

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
		return rumerrors.NewBadRequestError("INVALID_GROUP")
	}

	//private group is NOT supported
	if group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		return rumerrors.NewBadRequestError("FUNCTION_NOT_SUPPORTED")
	}

	ciperKey, err := hex.DecodeString(group.Item.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	decryptData, err := localcrypto.AesDecode(sendTrxItem.TrxItem, ciperKey)
	trxItem := new(NodeSDKTrxItem)

	if err := json.Unmarshal(decryptData, trxItem); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	if trxItem.JwtToken != NodeSDKJwtToken {
		return rumerrors.NewBadRequestError("INVALID_JWT_TOKEN")
	}

	trx := new(quorumpb.Trx)
	err = proto.Unmarshal(trxItem.TrxBytes, trx)
	if err != nil {
		return rumerrors.NewBadRequestError("INVALID_DATA")
	}

	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, nodectx.GetNodeCtx().Name)
	if err != nil {
		return rumerrors.NewBadRequestError("CHECK_AUTH_FAILED")
	}

	if !isAllow {
		return rumerrors.NewBadRequestError("OPERATION_DENY")
	}

	trxId, err := group.SendRawTrx(trx)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	sendTrxResult := &SendTrxResult{TrxId: trxId}
	return c.JSON(http.StatusOK, sendTrxResult)
}

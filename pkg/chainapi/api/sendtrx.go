package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type SendTrxResult struct {
	TrxId   string `json:"trx_id"   validate:"required"`
	ErrInfo string `json:"err_info" validate:"required"`
}

type NodeSDKTrxItem struct {
	TrxBytes []byte
	JwtToken string
}

type NodeSDKSendTrxItem struct {
	GroupId string
	TrxItem []byte
}

//discuss with wanming
func is_user_blocked(c echo.Context) bool {
	return false
}

func (h *Handler) SendTrx(c echo.Context) (err error) {

	if is_user_blocked(c) {
		return c.JSON(http.StatusForbidden, "")
	}

	output := make(map[string]string)
	sendTrxItem := new(NodeSDKSendTrxItem)
	if err = c.Bind(sendTrxItem); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[sendTrxItem.GroupId]; ok {
		//private group is NOT supported
		if group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			output[ERROR_INFO] = "FUNCTION_NOT_SUPPORTED"
			return c.JSON(http.StatusBadRequest, output)
		}

		ciperKey, err := hex.DecodeString(group.Item.CipherKey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		decryptData, err := localcrypto.AesDecode(sendTrxItem.TrxItem, ciperKey)
		trxItem := new(NodeSDKTrxItem)

		err = json.Unmarshal(decryptData, trxItem)
		if err != nil {
			output[ERROR_INFO] = "INVALID_DATA"
			return c.JSON(http.StatusBadRequest, output)
		}

		if trxItem.JwtToken != NodeSDKJwtToken {
			output[ERROR_INFO] = "INVALID_JWT_TOKEN"
			return c.JSON(http.StatusBadRequest, output)
		}

		trx := new(quorumpb.Trx)
		err = proto.Unmarshal(trxItem.TrxBytes, trx)
		if err != nil {
			output[ERROR_INFO] = "INVALID_DATA"
			return c.JSON(http.StatusBadRequest, output)
		}

		trxId, err := group.SendRawTrx(trx)
		var sendTrxResult *SendTrxResult
		if err != nil {
			sendTrxResult = &SendTrxResult{TrxId: trxId, ErrInfo: err.Error()}
			return c.JSON(http.StatusOK, sendTrxResult)
		} else {
			sendTrxResult = &SendTrxResult{TrxId: trxId, ErrInfo: "OK"}
			return c.JSON(http.StatusOK, sendTrxResult)
		}

	} else {
		output[ERROR_INFO] = "INVALID_GROUP"
		return c.JSON(http.StatusBadRequest, output)
	}
}

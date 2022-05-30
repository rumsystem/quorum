package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type GetGroupCtnPrarms struct {
	GroupId         string   `json:"group_id" validate:"required"`
	Num             int      `json:"num" validate:"required"`
	Nonce           string   `json:"nonce"`
	StartTrx        string   `json:"start_trx" validate:"required"`
	Reverse         string   `json:"reverse" validate:"required,oneof=true false"`
	IncludeStartTrx string   `json:"include_start_trx" validate:"required,oneof=true false"`
	Senders         []string `json:"senders"`
}

type GetGroupCtnItem struct {
	Req      GetGroupCtnPrarms
	JwtToken string
}

type GetGroupCtnReqItem struct {
	GroupId string
	Req     []byte
}

type NodeSdkGroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   []byte
	//TypeUrl   string
	TimeStamp int64
}

func (h *Handler) GetContentNSdk(c echo.Context) (err error) {
	if is_user_blocked(c) {
		return c.JSON(http.StatusForbidden, "")
	}

	output := make(map[string]string)
	getGroupCtnReqItem := new(GetGroupCtnReqItem)

	if err = c.Bind(getGroupCtnReqItem); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[getGroupCtnReqItem.GroupId]; ok {
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

		decryptData, err := localcrypto.AesDecode(getGroupCtnReqItem.Req, ciperKey)
		reqItem := new(GetGroupCtnItem)
		err = json.Unmarshal(decryptData, reqItem)
		if err != nil {
			output[ERROR_INFO] = "INVALID_DATA"
			return c.JSON(http.StatusBadRequest, output)
		}

		if reqItem.JwtToken != NodeSDKJwtToken {
			output[ERROR_INFO] = "INVALID_JWT_TOKEN"
			return c.JSON(http.StatusBadRequest, output)
		}

		num := reqItem.Req.Num //strconv.Atoi(reqItem.Num)
		nonce, _ := strconv.ParseInt(reqItem.Req.Nonce, 10, 64)
		starttrx := reqItem.Req.StartTrx
		if num == 0 {
			num = 20
		}
		reverse := false
		if reqItem.Req.Reverse == "true" {
			reverse = true
		}
		includestarttrx := false
		if reqItem.Req.IncludeStartTrx == "true" {
			includestarttrx = true
		}

		trxids, err := h.Appdb.GetGroupContentBySenders(getGroupCtnReqItem.GroupId,
			reqItem.Req.Senders,
			starttrx,
			nonce,
			num,
			reverse,
			includestarttrx)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		ctnobjList := []*NodeSdkGroupContentObjectItem{}
		for _, trxid := range trxids {
			trx, _, err := group.GetTrx(trxid.TrxId)
			if err != nil {
				c.Logger().Errorf("GetTrx Err: %s", err)
				continue
			}
			//don't decrypt trx data, nodesdk client will do it
			//huoju
			//how to handle typeurl
			ctnobjitem := &NodeSdkGroupContentObjectItem{TrxId: trx.TrxId, Publisher: trx.SenderPubkey, Content: trx.Data, TimeStamp: trx.TimeStamp}
			ctnobjList = append(ctnobjList, ctnobjitem)
		}
		return c.JSON(http.StatusOK, ctnobjList)
	} else {
		output[ERROR_INFO] = "INVALID_GROUP"
		return c.JSON(http.StatusBadRequest, output)
	}
}

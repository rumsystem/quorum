package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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
	Req GetGroupCtnPrarms
}

type GetGroupCtnReqItem struct {
	GroupId string `param:"group_id" validate:"required"`
	Req     []byte
}

func (h *Handler) GetContentNSdk(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	if is_user_blocked(c) {
		return c.JSON(http.StatusForbidden, "")
	}

	getGroupCtnReqItem := new(GetGroupCtnReqItem)
	if err := cc.BindAndValidate(getGroupCtnReqItem); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[getGroupCtnReqItem.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	}

	//private group is NOT supported
	if group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		return rumerrors.NewBadRequestError(rumerrors.ErrPrivateGroupNotSupported)
	}

	ciperKey, err := hex.DecodeString(group.Item.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	decryptData, err := localcrypto.AesDecode(getGroupCtnReqItem.Req, ciperKey)
	reqItem := new(GetGroupCtnItem)
	err = json.Unmarshal(decryptData, reqItem)
	if err != nil {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupData)
	}

	num := reqItem.Req.Num
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
		return rumerrors.NewBadRequestError(err)
	}

	trxList := []*quorumpb.Trx{}
	for _, trxid := range trxids {
		trx, _, err := group.GetTrx(trxid.TrxId)
		if err != nil {
			c.Logger().Errorf("GetTrx Err: %s", err)
			continue
		}
		trxList = append(trxList, trx)
	}
	return c.JSON(http.StatusOK, trxList)
}

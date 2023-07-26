package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GetGroupCtnPrarms struct {
	GroupId         string   `json:"group_id" validate:"required,uuid4"`
	Num             int      `json:"num" validate:"required"`
	StartTrx        string   `json:"start_trx" validate:"required"`
	Reverse         string   `json:"reverse" validate:"required,oneof=true false"`
	IncludeStartTrx string   `json:"include_start_trx" validate:"required,oneof=true false"`
	Senders         []string `json:"senders"`
}

type GetGroupCtnItem struct {
	Req GetGroupCtnPrarms
}

type GetGroupCtnReqItem struct {
	GroupId string `param:"group_id" json:"-" validate:"required" example:"630a545b-1ff4-4b9e-9a5d-bb13b6f6a629"`
	Req     []byte `json:"Req" validate:"required" swaggertype:"primitive,string"` // base64 encoded req data
}

type TrxItem struct {
	Trx    *quorumpb.Trx `json:"trx"`
	Status string        `json:"status"`
}

type GetGroupCtnResponse struct {
	TrxList []*TrxItem `json:"trx_list"`
}

// @Tags NodeAPI
// @Summary GetContentNSdk
// @Description get content
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param   get_content_params  body GetGroupCtnReqItem  true  "get group content params"
// @Success 200 {object} []quorumpb.Trx
// @Router  /api/v1/node/groupctn/{group_id} [post]
func (h *Handler) GetContentNSdk(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	getGroupCtnReqItem := new(GetGroupCtnReqItem)
	if err := cc.BindAndValidate(getGroupCtnReqItem); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[getGroupCtnReqItem.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
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
		num,
		reverse,
		includestarttrx)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	trxList := []*TrxItem{}
	for _, trxid := range trxids {
		trx, isOnChain, err := group.GetTrx(trxid)
		if err != nil {
			c.Logger().Errorf("GetTrx Err: %s", err)
			continue
		}

		trxItem := &TrxItem{
			Trx: trx,
		}

		if isOnChain {
			trxItem.Status = "onchain"
		} else {
			trxItem.Status = "offchain"
		}

		trxList = append(trxList, trxItem)
	}
	return c.JSON(http.StatusOK, trxList)
}

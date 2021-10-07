package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
)

type AnnounceResult struct {
	GroupId         string `json:"group_id"`
	AnnouncedPubkey string `json:"announced_pubkey"`
	Sign            string `json:"sign"`
	TrxId           string `json:"trx_id"`
	Action          string `json:"action"`
	Type            string `json:"type"`
}

type AnnounceParam struct {
	GroupId string `from:"group_id"      json:"group_id"      validate:"required"`
	Action  string `from:"action"      json:"action"      validate:"required,oneof=add del"`
	Type    string `from:"type"      json:"type"      validate:"required,oneof=userpubkey"`
}

// @Tags User
// @Summary AnnounceUserPubkey
// @Description Announce User's encryption Pubkey to the group
// @Accept json
// @Produce json
// @Param data body AnnounceParam true "AnnounceParam"
// @Success 200 {object} AnnounceResult
// @Router /api/v1/group/announce [post]
func (h *Handler) Announce(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(AnnounceParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var item *quorumpb.AnnounceItem
	item = &quorumpb.AnnounceItem{}
	item.GroupId = params.GroupId
	item.Action = params.Action
	item.Type = params.Type // "userpubkey"

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else {

		item.AnnouncedPubkey, err = nodectx.GetNodeCtx().Keystore.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.AnnouncedPubkey))
		buffer.Write([]byte(item.Type))
		buffer.Write([]byte(item.Action))
		hash := chain.Hash(buffer.Bytes())
		signature, err := nodectx.GetNodeCtx().Keystore.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.AnnouncerSignature = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()

		trxId, err := group.UpdAnnounce(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var announceResult *AnnounceResult
		announceResult = &AnnounceResult{GroupId: item.GroupId, AnnouncedPubkey: item.AnnouncedPubkey, Sign: hex.EncodeToString(signature), Type: item.Type, TrxId: trxId}

		return c.JSON(http.StatusOK, announceResult)
	}
}

package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type AnnounceResult struct {
	GroupId                string `json:"group_id"`
	AnnouncedSignPubkey    string `json:"sign_pubkey"`
	AnnouncedEncryptPubkey string `json:"encrypt_pubkey"`
	Type                   string `json:"type"`
	Action                 string `json:"action"`
	Sign                   string `json:"sign"`
	TrxId                  string `json:"trx_id"`
}

type AnnounceParam struct {
	GroupId string `from:"group_id"    json:"group_id"    validate:"required"`
	Action  string `from:"action"      json:"action"      validate:"required,oneof=add remove"`
	Type    string `from:"type"        json:"type"        validate:"required,oneof=user producer"`
	Memo    string `from:"memo"        json:"memo"        validate:"required"`
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

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		if params.Type == "user" {
			item.Type = quorumpb.AnnounceType_AS_USER
		} else if params.Type == "producer" {
			item.Type = quorumpb.AnnounceType_AS_PRODUCER
		} else {
			output[ERROR_INFO] = "Unknown type"
			return c.JSON(http.StatusBadRequest, output)
		}

		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			output[ERROR_INFO] = "Unknown action"
			return c.JSON(http.StatusBadRequest, output)
		}

		item.SignPubkey = group.Item.UserSignPubkey

		if item.Type == quorumpb.AnnounceType_AS_USER {
			item.EncryptPubkey, err = nodectx.GetNodeCtx().Keystore.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		}

		item.OwnerPubkey = ""
		item.OwnerSignature = ""
		item.Result = quorumpb.ApproveType_ANNOUNCED

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.SignPubkey))
		buffer.Write([]byte(item.EncryptPubkey))
		buffer.Write([]byte(item.Type.String()))
		hash := chain.Hash(buffer.Bytes())
		signature, err := nodectx.GetNodeCtx().Keystore.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.AnnouncerSignature = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		item.Memo = params.Memo

		trxId, err := group.UpdAnnounce(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var announceResult *AnnounceResult
		announceResult = &AnnounceResult{GroupId: item.GroupId, AnnouncedSignPubkey: item.SignPubkey, AnnouncedEncryptPubkey: item.EncryptPubkey, Type: item.Type.String(), Action: item.Action.String(), Sign: hex.EncodeToString(signature), TrxId: trxId}

		return c.JSON(http.StatusOK, announceResult)
	}
}

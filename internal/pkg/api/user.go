package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type GrpUserResult struct {
	GroupId       string `json:"group_id"`
	UserPubkey    string `json:"user_pubkey"`
	EncryptPubkey string `json:"encrypt_pubkey"`
	OwnerPubkey   string `json:"owner_pubkey"`
	Sign          string `json:"sign"`
	TrxId         string `json:"trx_id"`
	Memo          string `json:"memo"`
	Action        string `json:"action"`
}

type GrpUserParam struct {
	Action     string `from:"action"          json:"action"           validate:"required,oneof=add remove"`
	UserPubkey string `from:"user_pubkey" json:"user_pubkey"  validate:"required"`
	GroupId    string `from:"group_id"        json:"group_id"         validate:"required"`
	Memo       string `from:"memo"            json:"memo"`
}

// @Tags Management
// @Summary AddUsers
// @Description add a user to a private group users list
// @Accept json
// @Produce json
// @Param data body GrpUserParam true "GrpUserParam"
// @Success 200 {object} GrpUserResult
// @Router /api/v1/group/user [post]
func (h *Handler) GroupUser(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(GrpUserParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove user"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		isAnnounced, err := group.IsUserAnnounced(params.UserPubkey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if !isAnnounced {
			output[ERROR_INFO] = "User is not announced"
			return c.JSON(http.StatusBadRequest, output)
		}

		user, err := group.GetAnnouncedUser(params.UserPubkey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if user.Action == quorumpb.ActionType_REMOVE && params.Action == "add" {
			output[ERROR_INFO] = "Can not add a none active user"
			return c.JSON(http.StatusBadRequest, output)
		}

		if user.Result == quorumpb.ApproveType_ANNOUNCED && params.Action == "remove" {
			output[ERROR_INFO] = "Can not remove an unapprove user"
			return c.JSON(http.StatusBadRequest, output)
		}

		if user.Result == quorumpb.ApproveType_APPROVED && params.Action == "add" {
			output[ERROR_INFO] = "Can not add an approved user"
			return c.JSON(http.StatusBadRequest, output)
		}

		item := &quorumpb.UserItem{}
		item.GroupId = params.GroupId
		item.UserPubkey = params.UserPubkey
		item.EncryptPubkey = user.EncryptPubkey
		item.GroupOwnerPubkey = group.Item.OwnerPubKey

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.UserPubkey))
		buffer.Write([]byte(item.EncryptPubkey))
		buffer.Write([]byte(item.GroupOwnerPubkey))
		hash := chain.Hash(buffer.Bytes())

		ks := nodectx.GetNodeCtx().Keystore
		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			output[ERROR_INFO] = "Unknown action"
			return c.JSON(http.StatusBadRequest, output)
		}

		item.Memo = params.Memo
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdUser(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var blockGrpUserResult *GrpUserResult
		blockGrpUserResult = &GrpUserResult{GroupId: item.GroupId, UserPubkey: item.UserPubkey, EncryptPubkey: item.EncryptPubkey, OwnerPubkey: item.GroupOwnerPubkey, Sign: item.GroupOwnerSign, Action: item.Action.String(), Memo: item.Memo, TrxId: trxId}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}
}

package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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
	Action     string `from:"action"          json:"action"       validate:"required,oneof=add remove"`
	UserPubkey string `from:"user_pubkey"     json:"user_pubkey"  validate:"required"`
	GroupId    string `from:"group_id"        json:"group_id"     validate:"required"`
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
	cc := c.(*utils.CustomContext)

	params := new(GrpUserParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return rumerrors.NewBadRequestError(rumerrors.ErrOnlyGroupOwner)
	} else {
		isAnnounced, err := h.ChainAPIdb.IsUserAnnounced(group.Item.GroupId, params.UserPubkey, group.ChainCtx.GetNodeName())
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if !isAnnounced {
			return rumerrors.NewBadRequestError("User is not announced")
		}

		user, err := group.GetAnnouncedUser(params.UserPubkey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if user.Action == quorumpb.ActionType_REMOVE && params.Action == "add" {
			return rumerrors.NewBadRequestError("Can not add a none active user")
		}

		if user.Result == quorumpb.ApproveType_ANNOUNCED && params.Action == "remove" {
			return rumerrors.NewBadRequestError("Can not remove an unapprove user")
		}

		if user.Result == quorumpb.ApproveType_APPROVED && params.Action == "add" {
			return rumerrors.NewBadRequestError("Can not add an approved user")
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
		hash := localcrypto.Hash(buffer.Bytes())

		ks := nodectx.GetNodeCtx().Keystore
		signature, err := ks.EthSignByKeyName(item.GroupId, hash)

		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			return rumerrors.NewBadRequestError("Unknown action")
		}

		item.Memo = params.Memo
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdUser(item)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		blockGrpUserResult := &GrpUserResult{
			GroupId:       item.GroupId,
			UserPubkey:    item.UserPubkey,
			EncryptPubkey: item.EncryptPubkey,
			OwnerPubkey:   item.GroupOwnerPubkey,
			Sign:          item.GroupOwnerSign,
			Action:        item.Action.String(),
			Memo:          item.Memo,
			TrxId:         trxId,
		}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}
}

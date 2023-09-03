package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type UpdGrpSyncerResult struct {
	GroupId      string `from:"group_id"        json:"group_id" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	SyncerPubkey string `from:"syncer_pubkey"   json:"syncer_pubkey" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	Action       string `json:"action" example:"ADD"`
	Memo         string `json:"memo"`
	TrxId        string `json:"trx_id"`
}

type UpdGrpSyncerParam struct {
	GroupId      string `from:"group_id"        json:"group_id"     validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	SyncerPubkey string `from:"syncer_pubkey"   json:"syncer_pubkey" validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	Action       string `from:"action"          json:"action"       validate:"required,oneof=add remove" example:"add"`
	Memo         string `from:"memo"            json:"memo"`
}

// @Tags Management
// @Summary UpdGroupUser
// @Description add or remove a user(pubkey) to/from a private group
// @Accept json
// @Produce json
// @Param data body UpdGrpUserParam true "UpdGrpUserParam"
// @Success 200 {object} UpdGrpUserResult
// @Router /api/v1/group/upduser [post]
func (h *Handler) UpdGroupSyncer(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	params := new(UpdGrpSyncerParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	} else if group.Item.SyncType == quorumpb.GroupSyncType_PUBLIC {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotPrivate)
	} else {
		var action quorumpb.ActionType
		if params.Action == "add" {
			action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			action = quorumpb.ActionType_REMOVE
		} else {
			return rumerrors.NewBadRequestError("Invalid action")
		}

		syncer := &quorumpb.Syncer{
			GroupId:      params.GroupId,
			SyncerPubkey: params.SyncerPubkey,
			Memo:         params.Memo,
		}

		item := &quorumpb.UpdGroupSyncerItem{
			Syncer: syncer,
			Action: action,
			Memo:   params.Memo,
		}

		trxId, err := group.UpdGroupSyncer(item)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		result := &UpdGrpSyncerResult{
			GroupId:      params.GroupId,
			SyncerPubkey: params.SyncerPubkey,
			Action:       item.Action.String(),
			Memo:         item.Memo,
			TrxId:        trxId,
		}

		return c.JSON(http.StatusOK, result)
	}
}

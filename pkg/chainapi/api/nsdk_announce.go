package api

import (
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type (
	NSdkAnnounceParams struct {
		GroupId string                 `param:"group_id" json:"-" validate:"required"`
		Data    *quorumpb.AnnounceItem `json:"data" validate:"required"`
	}
)

// @Tags LightNode
// @Summary NSdkAnnounce
// @Description Announce User's encryption Pubkey to the group for light node
// @Accept json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param data body NSdkAnnounceParams true "AnnounceParam"
// @Success 200 {object} handlers.AnnounceResult
// @Router /api/v1/node/{group_id}/announce [post]
func (h *Handler) NSdkAnnounce(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	payload := new(NSdkAnnounceParams)
	if err := cc.BindAndValidate(payload); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[payload.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError("INVALID_GROUP")
	}

	item := payload.Data
	trxId, err := group.UpdAnnounce(item)
	if err != nil {
		return err
	}
	var result *handlers.AnnounceResult
	result = &handlers.AnnounceResult{
		GroupId:       item.GroupId,
		SignPubkey:    item.Content.SignPubkey,
		EncryptPubkey: item.Content.EncryptPubkey,
		Type:          item.Content.Type.String(),
		Action:        item.Action.String(),
		Sign:          hex.EncodeToString(item.Signature),
		TrxId:         trxId,
	}

	return c.JSON(http.StatusOK, result)
}

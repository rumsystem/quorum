package api

import (
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type (
	AnnounceNodeSDKParam struct {
		GroupId string `param:"group_id" json:"-" validate:"required"`
		Req     []byte `json:"Req" validate:"required" swaggertype:"primitive,string"` // base64 encoded req
	}
)

// @Tags LightNode
// @Summary AnnounceUserPubkey
// @Description Announce User's encryption Pubkey to the group for light node
// @Accept json
// @Produce json
// @Param data body AnnounceNodeSDKParam true "AnnounceParam"
// @Success 200 {object} handlers.AnnounceResult
// @Router /api/v1/node/announce/{group_id} [post]
func (h *Handler) AnnounceNodeSDK(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(AnnounceNodeSDKParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError("INVALID_GROUP")
	}

	ciperKey, err := hex.DecodeString(group.Item.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError("CHAINSDK_INTERNAL_ERROR")
	}

	decryptData, err := localcrypto.AesDecode(params.Req, ciperKey)
	if err != nil {
		return rumerrors.NewBadRequestError("DECRYPT_DATA_FAILED")
	}

	var item quorumpb.AnnounceItem
	if err := proto.Unmarshal(decryptData, &item); err != nil {
		return rumerrors.NewBadRequestError("Req is not quorumpb.AnnounceItem object")
	}

	trxId, err := group.UpdAnnounce(&item)
	if err != nil {
		return err
	}

	result := &handlers.AnnounceResult{
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

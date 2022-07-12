package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetGroupByIdParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) GetGroupById() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error

		groupid := c.Param("group_id")
		if groupid == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
		}

		groupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		groupInfo := &GroupInfo{}
		groupInfo.GroupId = groupItem.Group.GroupId
		groupInfo.GroupName = groupItem.Group.GroupName
		groupInfo.SignAlias = groupItem.SignAlias
		groupInfo.EncryptAlias = groupItem.EncryptAlias

		ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(groupItem.Group.UserSignPubkey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		groupInfo.UserEthaddr = ethaddr
		groupInfo.ConsensusType = groupItem.Group.ConsenseType.String()
		groupInfo.EncryptionType = groupItem.Group.EncryptType.String()
		groupInfo.CipherKey = groupItem.Group.CipherKey
		groupInfo.AppKey = groupItem.Group.AppKey
		groupInfo.LastUpdated = groupItem.Group.LastUpdate
		groupInfo.HighestHeight = groupItem.Group.HighestHeight
		groupInfo.HighestBlockId = groupItem.Group.HighestBlockId
		groupInfo.ChainApis = groupItem.ApiUrl

		return c.JSON(http.StatusOK, groupInfo)
	}
}

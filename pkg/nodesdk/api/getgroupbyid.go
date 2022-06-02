package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetGroupByIdParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) GetGroupById() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		groupid := c.Param("group_id")
		if groupid == "" {
			output[ERROR_INFO] = "group_id can not be empty"
			return c.JSON(http.StatusBadRequest, output)
		}

		dbMgr := nodesdkctx.GetDbMgr()
		groupItem, err := dbMgr.GetGroupInfo(groupid)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var groupInfo *GroupInfo
		groupInfo = &GroupInfo{}
		groupInfo.GroupId = groupItem.Group.GroupId
		groupInfo.GroupName = groupItem.Group.GroupName
		groupInfo.SignAlias = groupItem.SignAlias
		groupInfo.EncryptAlias = groupItem.EncryptAlias

		ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(groupItem.Group.UserSignPubkey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
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

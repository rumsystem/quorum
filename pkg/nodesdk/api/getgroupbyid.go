package nodesdkapi

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetGroupByIdParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) GetGroupById() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		validate := validator.New()
		params := new(GetGroupByIdParams)
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		dbMgr := nodesdkctx.GetDbMgr()
		groupItem, err := dbMgr.GetGroupInfo(params.GroupId)
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

		/*
			Check with huoju
			ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(groupItem.Group.UserSignPubkey)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		*/

		groupInfo.UserEthaddr = groupItem.Group.UserSignPubkey
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

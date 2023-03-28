package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type GetDataNodeSDKItem struct {
	GroupId string `param:"group_id" json:"-" validate:"required"`
	ReqType string `json:"ReqType" validate:"required,oneof=auth_type auth_allowlist auth_denylist appconfig_listlist appconfig_item_bykey announced_producer announced_user group_producer group_info"`
	Req     []byte `json:"Req" validate:"required" swaggertype:"primitive,string"` // base64 encoded req
}

type GrpInfo struct {
	GroupId string
}

type (
	GetNSdkGroupInfoParams struct {
		GroupId string `param:"group_id" json:"group_id" validate:"required,uuid4"`
	}
)

type AppConfigKeyListItem struct {
	GroupId string
}

type AppConfigItem struct {
	GroupId string
	Key     string
}

type AnnGrpProducer struct {
	GroupId string
}

type GrpProducer struct {
	GroupId string
}

type AnnGrpUser struct {
	GroupId    string
	SignPubkey string
}

type GrpInfoNodeSDK struct {
	GroupId      string `json:"group_id"`
	Owner        string `json:"owner"`
	LatestUpdate int64  `json:"latest_update"`
	Provider     string `json:"provider"`
	Singature    string `json:"singature"`
}

// @Tags LightNode
// @Summary GetNSdkGroupInfo
// @Description get group info from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} []GrpInfoNodeSDK
// @Router  /api/v1/node/{group_id}/info [get]
func (h *Handler) GetNSdkGroupInfo(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(GetNSdkGroupInfoParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	groupmgr := chain.GetGroupMgr()
	if grp, ok := groupmgr.Groups[params.GroupId]; ok {
		grpInfo := new(GrpInfoNodeSDK)
		grpInfo.GroupId = grp.Item.GroupId
		grpInfo.Owner = grp.Item.OwnerPubKey
		grpInfo.Provider = grp.Item.UserSignPubkey
		grpInfo.LatestUpdate = grp.Item.LastUpdate

		return c.JSON(http.StatusOK, grpInfo)
	} else {
		return rumerrors.NewBadRequestError("INVALID_GROUP")
	}
}

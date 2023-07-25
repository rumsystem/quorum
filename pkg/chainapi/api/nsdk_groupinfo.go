package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type (
	GetNSdkGroupInfoParams struct {
		GroupId string `param:"group_id" json:"group_id" validate:"required,uuid4"`
	}
)

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

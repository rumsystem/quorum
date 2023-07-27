package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type (
	GetNSdkUserEncryptPubKeysParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4" example:"78cbab65-17e7-49d2-892a-311cec77c120"`
	}

	GetUserEncryptPubKeysResult struct {
		Keys []string `json:"keys" example:"age1gcd6v44ys4u72ljc543er65sj8qlscnwqp2nm4m9yg7zwcc0648q7swrka,age1fxfkenckddacqpm9ar3wvyg4ek32p9d7rlyz28y4catzfhjw4ggs8fvdl5"`
	}
)

// @Tags LightNode
// @Summary GetNSdkUserEncryptPubKeys
// @Description get user encrypt pub keys
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} GetUserEncryptPubKeysResult
// @Router  /api/v1/node/{group_id}/userencryptpubkeys [get]
func (h *Handler) GetNSdkUserEncryptPubKeys(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	param := GetNSdkUserEncryptPubKeysParams{}
	if err := cc.BindAndValidate(&param); err != nil {
		return err
	}
	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[param.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError("INVALID_GROUP")
	}

	keys, err := group.ChainCtx.GetUserEncryptPubKeys()
	if err != nil {
		return err
	}

	result := GetUserEncryptPubKeysResult{Keys: keys}
	return c.JSON(http.StatusOK, result)
}

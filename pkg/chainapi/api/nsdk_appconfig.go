package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type (
	GetNSdkAppconfigKeylistParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4"`
	}

	GetNSdkAppconfigByKeyParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4"`
		Key     string `param:"key" json:"key" url:"key" validate:"required"`
	}
)

// @Tags LightNode
// @Summary GetNSdkAppconfigKeylist
// @Description get app config key list from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} []handlers.AppConfigKeyListItem
// @Router  /api/v1/node/{group_id}/appconfig/keylist [get]
func (h *Handler) GetNSdkAppconfigKeylist(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	params := new(GetNSdkAppconfigKeylistParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	res, err := handlers.GetAppConfigKeyList(params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

// @Tags LightNode
// @Summary GetNSdkAppconfigByKey
// @Description get app config by key from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param   key path string true "key name"
// @Success 200 {object} []handlers.AppConfigKeyItem
// @Router  /api/v1/node/{group_id}/appconfig/by/{key} [get]
func (h *Handler) GetNSdkAppconfigByKey(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(GetNSdkAppconfigByKeyParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	res, err := handlers.GetAppConfigByKey(params.Key, params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

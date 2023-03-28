package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type (
	GetNSdkAuthTypeParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4"`
		TrxType string `param:"trx_type" json:"trx_type" url:"trx_type" validate:"required,oneof=POST ANNOUNCE REQ_BLOCK"`
	}

	GetNSdkAllowListParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4"`
	}

	GetNSdkDenyListParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4"`
	}
)

// @Tags LightNode
// @Summary GetNSdkAuthType
// @Description get auth type from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param   trx_type path string true "trx type"
// @Success 200 {object} handlers.TrxAuthItem
// @Router  /api/v1/node/{group_id}/auth/by/{trx_type} [get]
func (h *Handler) GetNSdkAuthType(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	params := new(GetNSdkAuthTypeParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	res, err := handlers.GetChainTrxAuthMode(h.ChainAPIdb, params.GroupId, params.TrxType)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

// @Tags LightNode
// @Summary GetNSdkAllowList
// @Description get allow list from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} []handlers.ChainSendTrxRuleListItem
// @Router  /api/v1/node/{group_id}/auth/alwlist [get]
func (h *Handler) GetNSdkAllowList(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	params := new(GetNSdkAllowListParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	res, err := handlers.GetChainTrxAllowList(h.ChainAPIdb, params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

// @Tags LightNode
// @Summary GetNSdkDenyList
// @Description get deny list from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} []handlers.ChainSendTrxRuleListItem
// @Router  /api/v1/node/{group_id}/auth/denylist [get]
func (h *Handler) GetNSdkDenyList(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	params := new(GetNSdkDenyListParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	res, err := handlers.GetChainTrxDenyList(h.ChainAPIdb, params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type (
	GetNSdkAnnouncedProducerParams struct {
		GroupId string `param:"group_id" json:"group_id" url:"group_id" validate:"required,uuid4"`
	}

	GetNSdkAnnouncedUserParams struct {
		GroupId string `param:"group_id" json:"-" validate:"required,uuid4"`
		PubKey  string `query:"pubkey" json:"pubkey"`
	}

	GetNSdkGroupProducersParams struct {
		GroupId string `param:"group_id" json:"group_id" validate:"required,uuid4"`
	}
)

// @Tags LightNode
// @Summary GetNSdkAnnouncedProducer
// @Description get announced producer from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} []handlers.AppConfigKeyItem
// @Router  /api/v1/node/{group_id}/announced/producer [get]
func (h *Handler) GetNSdkAnnouncedProducer(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(GetNSdkAnnouncedProducerParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	res, err := handlers.GetAnnouncedGroupProducer(h.ChainAPIdb, params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

// @Tags LightNode
// @Summary GetNSdkAnnouncedUser
// @Description get announced user from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param   get_announced_user_params  query GetNSdkAnnouncedUserParams true  "get announced user params"
// @Success 200 {object} []handlers.AnnouncedUserListItem
// @Router  /api/v1/node/{group_id}/announced/user [get]
func (h *Handler) GetNSdkAnnouncedUser(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(GetNSdkAnnouncedUserParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	var res = []*handlers.AnnouncedUserListItem{}
	if params.PubKey == "" {
		res, err = handlers.GetAnnouncedGroupUsers(h.ChainAPIdb, params.GroupId)
	} else {
		item, err := handlers.GetAnnouncedGroupUser(params.GroupId, params.PubKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		res = append(res, item)
	}
	return c.JSON(http.StatusOK, res)
}

// @Tags LightNode
// @Summary GetNSdkGroupProducers
// @Description get group producers from chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Success 200 {object} []handlers.ProducerListItem
// @Router  /api/v1/node/{group_id}/producers [get]
func (h *Handler) GetNSdkGroupProducers(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(GetNSdkGroupProducersParams)
	if err := cc.BindAndValidate(params); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	res, err := handlers.GetGroupProducers(h.ChainAPIdb, params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	return c.JSON(http.StatusOK, res)
}

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

package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type LeaveGroupParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

type LeaveGroupResult struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) LeaveGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(LeaveGroupParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		// save nodesdkgroupitem to db
		if err := nodesdkctx.GetCtx().GetChainStorage().RmGroup(params.GroupId); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		leaveGroupResult := &LeaveGroupResult{GroupId: params.GroupId}

		return c.JSON(http.StatusOK, leaveGroupResult)
	}
}

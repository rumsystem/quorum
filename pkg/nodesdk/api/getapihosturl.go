package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	pkgutils "github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetApiHostParams struct {
	GroupId string `param:"group_id" validate:"required"`
}

type GetApiHostResult struct {
	URLs []string `json:"urls" validate:"required"`
}

func (h *NodeSDKHandler) GetApiHostUrl(c echo.Context) (err error) {
	cc := c.(*pkgutils.CustomContext)
	params := new(GetApiHostParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	result := GetApiHostResult{URLs: nodesdkGroupItem.ApiUrl}

	return c.JSON(http.StatusOK, result)
}

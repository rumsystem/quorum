package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	pkgutils "github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type UpdApiHostUrlParams struct {
	GroupId      string   `json:"group_id" validate:"required"`
	ChainAPIUrls []string `json:"urls" validate:"required"`
}

func (h *NodeSDKHandler) UpdApiHostUrl(c echo.Context) (err error) {
	cc := c.(*pkgutils.CustomContext)
	params := new(UpdApiHostUrlParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	nodesdkGroupItem.ApiUrl = params.ChainAPIUrls

	if err := nodesdkctx.GetCtx().GetChainStorage().UpdGroupV2(nodesdkGroupItem); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, "")
}

package nodesdkapi

import (
	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	pkgutils "github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type UpdApiHostUrlParams struct {
	GroupId      string   `json:"group_id" validate:"required"`
	ChainAPIUrls []string `json:"urls" validate:"required,gte=1,unique,dive,required,url"`
}

func (h *NodeSDKHandler) UpdApiHostUrl(c echo.Context) (err error) {
	cc := c.(*pkgutils.CustomContext)
	params := new(UpdApiHostUrlParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	for _, _url := range params.ChainAPIUrls {
		_, jwt, err := utils.ParseChainapiURL(_url)
		if err != nil {
			return rumerrors.NewBadRequestError("invalid chain api url")
		}
		if jwt == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidJWT)
		}
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	nodesdkGroupItem.ApiUrl = params.ChainAPIUrls

	if err := nodesdkctx.GetCtx().GetChainStorage().UpdGroupV2(nodesdkGroupItem); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return cc.Success()
}

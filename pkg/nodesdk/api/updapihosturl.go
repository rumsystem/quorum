package nodesdkapi

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type UpdApiHostUrlParams struct {
	GroupId     string   `json:"group_id" validate:"required"`
	ChainAPIUrl []string `json:"urls"     validate:"required"`
}

func (h *NodeSDKHandler) UpdApiHostUrl(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(UpdApiHostUrlParams)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(params.GroupId)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	nodesdkGroupItem.ApiUrl = params.ChainAPIUrl

	err = nodesdkctx.GetCtx().GetChainStorage().UpdGroupV2(nodesdkGroupItem)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, "")
}

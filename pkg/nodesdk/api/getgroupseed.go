package nodesdkapi

import (
	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	"net/http"
)

type GetGroupSeedParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) GetGroupSeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)
		groupid := c.Param("group_id")

		if groupid == "" {
			output[ERROR_INFO] = "group_id can not be empty"
			return c.JSON(http.StatusBadRequest, output)
		}

		groupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, groupItem.GroupSeed)
	}
}

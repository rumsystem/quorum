package nodesdkapi

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
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

		dbMgr := nodesdkctx.GetDbMgr()
		groupItem, err := dbMgr.GetGroupInfo(groupid)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		fmt.Println(groupItem.GroupSeed)
		return c.JSON(http.StatusOK, groupItem.GroupSeed)
	}
}

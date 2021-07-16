package api

import (
	"fmt"
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type StartSyncResult struct {
	GroupId string
	Error   string
}

func (h *Handler) StartSync(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetChainCtx().Groups[groupid]; ok {

		if group.Status == chain.GROUP_DIRTY {
			error_info := "GROUP_ALREADY_IN_SYNCING"
			startSyncResult := &StartSyncResult{GroupId: group.Item.GroupId, Error: error_info}
			return c.JSON(http.StatusBadRequest, startSyncResult)
		} else {
			group.StartSync()
			startSyncResult := &StartSyncResult{GroupId: group.Item.GroupId, Error: ""}
			return c.JSON(http.StatusOK, startSyncResult)
		}
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

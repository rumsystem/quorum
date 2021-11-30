package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type StartSyncResult struct {
	GroupId string `validate:"required"`
	Error   string
}

// @Tags Group
// @Summary StartSync
// @Description Start sync
// @Produce json
// @Success 200 {object} StartSyncResult
// @Router /api/v1/group/{group_id}/startsync [post]
func (h *Handler) StartSync(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[groupid]
	if !ok {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}

	if group.ChainCtx.Syncer.Status == chain.SYNCING_BACKWARD || group.ChainCtx.Syncer.Status == chain.SYNCING_FORWARD {
		errorInfo := "GROUP_ALREADY_IN_SYNCING"
		startSyncResult := &StartSyncResult{GroupId: group.Item.GroupId, Error: errorInfo}
		return c.JSON(http.StatusBadRequest, startSyncResult)
	}

	startSyncResult := &StartSyncResult{GroupId: group.Item.GroupId, Error: ""}
	if err := group.StartSync(); err != nil {
		startSyncResult.Error = err.Error()
	}
	return c.JSON(http.StatusOK, startSyncResult)
}

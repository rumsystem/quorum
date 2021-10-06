package api

import (
	"fmt"
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type DeniedUserListItem struct {
	GroupId          string
	PeerId           string
	GroupOwnerPubkey string
	GroupOwnerSign   string
	TimeStamp        int64
	Action           string
	Memo             string
}

// @Tags Management
// @Summary GetDeniedUserList
// @Description Get the list of denied users
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} DeniedUserListItem
// @Router /api/v1/trx/{group_id}/deniedlist [get]
func (h *Handler) GetDeniedUserList(c echo.Context) (err error) {
	output := make(map[string]string)
	var result []*DeniedUserListItem

	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		blkList, err := group.GetBlockedUser()
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		for _, blkItem := range blkList {
			var item *DeniedUserListItem
			item = &DeniedUserListItem{}

			item.GroupId = blkItem.GroupId
			item.PeerId = blkItem.PeerId
			item.GroupOwnerPubkey = blkItem.GroupOwnerPubkey
			item.GroupOwnerSign = blkItem.GroupOwnerSign
			item.Action = blkItem.Action
			item.Memo = blkItem.Memo
			item.TimeStamp = blkItem.TimeStamp
			result = append(result, item)
		}
		return c.JSON(http.StatusOK, result)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

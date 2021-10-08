package api

import (
	"fmt"
	"net/http"

	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type AnnouncedUserListItem struct {
	UserPubkey string
}

// @Tags User
// @Summary GetAnnouncedGroupUsers
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} AnnouncedUserListItem
// @Router /api/v1/group/{group_id}/announced/users [get]
func (h *Handler) GetAnnouncedGroupUsers(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetNodeCtx().Groups[groupid]; ok {
		usrList, err := chain.GetDbMgr().GetAnnouncedUsers(group.Item.GroupId, chain.GetNodeCtx().Name)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		usrResultList := []*AnnouncedUserListItem{}
		for _, usr := range usrList {
			var item *AnnouncedUserListItem
			item = &AnnouncedUserListItem{}
			item.UserPubkey = usr.AnnouncedPubkey
			usrResultList = append(usrResultList, item)
		}

		return c.JSON(http.StatusOK, usrResultList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

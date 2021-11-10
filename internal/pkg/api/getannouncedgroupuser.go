package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/chain"
)

type AnnouncedUserListItem struct {
	AnnouncedSignPubkey    string
	AnnouncedEncryptPubkey string
	AnnouncerSign          string
	Result                 string
}

// @Tags User
// @Summary GetAnnouncedGroupUsers
// @Description Get the list of private group users
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

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		usrList, err := group.GetAnnouncedUsers()
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		usrResultList := []*AnnouncedUserListItem{}
		for _, usr := range usrList {
			var item *AnnouncedUserListItem
			item = &AnnouncedUserListItem{}
			item.AnnouncedSignPubkey = usr.SignPubkey
			item.AnnouncedEncryptPubkey = usr.EncryptPubkey
			item.AnnouncerSign = usr.AnnouncerSignature
			item.Result = usr.Result.String()
			usrResultList = append(usrResultList, item)
		}

		return c.JSON(http.StatusOK, usrResultList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

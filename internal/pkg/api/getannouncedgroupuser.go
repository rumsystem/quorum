package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags User
// @Summary GetAnnouncedGroupUsers
// @Description Get the list of private group users
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} handlers.AnnouncedUserListItem
// @Router /api/v1/group/{group_id}/announced/users [get]
func (h *Handler) GetAnnouncedGroupUsers(c echo.Context) error {
	output := make(map[string]string)
	groupid := c.Param("group_id")

	res, err := handlers.GetAnnouncedGroupUsers(groupid)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)

}

// @Tags User
// @Summary GetAnnouncedGroupUser
// @Description Get the one user announce status
// @Produce json
// @Param group_id path string true "Group Id"
// @Param sign_pubkey path string true "User SignPubkey"
// @Success 200 {object} handlers.AnnouncedUserListItem
// @Router /api/v1/group/{group_id}/announced/user/:sign_pubkey [get]
func (h *Handler) GetAnnouncedGroupUser(c echo.Context) error {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	sign_pubkey := c.Param("sign_pubkey")

	res, err := handlers.GetAnnouncedGroupUser(groupid, sign_pubkey)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)

}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags User
// @Summary GetAnnouncedGroupUsers
// @Description Get the list of private group users
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} handlers.AnnouncedUserListItem
// @Router /api/v1/group/{group_id}/announced/users [get]
func (h *Handler) GetAnnouncedGroupUsers(c echo.Context) error {
	groupid := c.Param("group_id")

	res, err := handlers.GetAnnouncedGroupUsers(h.ChainAPIdb, groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
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
// @Router /api/v1/group/{group_id}/announced/user/{sign_pubkey} [get]
func (h *Handler) GetAnnouncedGroupUser(c echo.Context) error {
	groupid := c.Param("group_id")
	sign_pubkey := c.Param("sign_pubkey")

	res, err := handlers.GetAnnouncedGroupUser(groupid, sign_pubkey)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

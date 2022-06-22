package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary Get group seed
// @Description get group seed from appdb
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {object} handlers.GroupSeed
// @Router /api/v1/group/{group_id}/seed [get]
func (h *Handler) GetGroupSeedHandler(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	if groupId == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "group_id can't be nil."})
	}

	seed, err := handlers.GetGroupSeed(groupId, h.Appdb)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("get group seed failed: %s", err)})
	}
	seedurl, err := handlers.GroupSeedToUrl(1, []string{}, seed)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("seedurl output failed: %s", err)})
	}

	output := make(map[string]string)
	output["seed"] = seedurl
	return c.JSON(http.StatusOK, output)
}

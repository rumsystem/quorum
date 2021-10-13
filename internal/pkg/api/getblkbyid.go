package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	_ "github.com/rumsystem/quorum/internal/pkg/pb" //import for swaggo
)

// @Tags Chain
// @Summary GetBlock
// @Description Get a block from a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param block_id path string  true "Block Id"
// @Success 200 {object} pb.Block
// @Router /api/v1/block/{group_id}/{block_id} [get]
func (h *Handler) GetBlockById(c echo.Context) (err error) {
	output := make(map[string]string)

	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	blockid := c.Param("block_id")
	if blockid == "" {
		output[ERROR_INFO] = "block_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		block, err := group.GetBlock(blockid)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, block)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

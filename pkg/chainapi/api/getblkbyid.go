package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	_ "github.com/rumsystem/rumchaindata/pkg/pb" //import for swaggo
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
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError("group_id can't be nil.")
	}

	blockid := c.Param("block_id")
	if blockid == "" {
		return rumerrors.NewBadRequestError("block_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		block, err := group.GetBlock(blockid)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, block)
	} else {
		return rumerrors.NewBadRequestError(fmt.Sprintf("Group %s not exist", groupid))
	}
}

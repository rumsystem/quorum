package api

import (
	"fmt"
	"net/http"
	"strconv"

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

	epoch := c.Param("epoch")
	// verify epoch is valid
	/*
		if epoch < 0  {
			return rumerrors.NewBadRequestError("block_id can't be nil.")
		}
	*/
	epocnInt64, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		block, err := group.GetBlock(epocnInt64)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, block)
	} else {
		return rumerrors.NewBadRequestError(fmt.Sprintf("Group %s not exist", groupid))
	}
}

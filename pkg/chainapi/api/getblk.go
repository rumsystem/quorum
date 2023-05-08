package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/pb"
	_ "github.com/rumsystem/quorum/pkg/pb" //import for swaggo
)

type GetBlockResponse struct {
	Block  *pb.Block `json:"block"`
	Status string    `json:"status"`
}

// @Tags Chain
// @Summary GetBlock
// @Description Get a block from a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param block_id path string  true "Epoch"
// @Success 200 {object} pb.Block
// @Router /api/v1/block/{group_id}/{epoch} [get]
func (h *Handler) GetBlock(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	blockIdStr := c.Param("block_id")
	if blockIdStr == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidBlockID)
	}

	blockId, err := strconv.ParseUint(blockIdStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		block, isOnChain, err := group.GetBlock(blockId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		resp := &GetBlockResponse{
			Block: block,
		}

		if isOnChain {
			resp.Status = "onchain"
		} else {
			resp.Status = "offchain"
		}
		return c.JSON(http.StatusOK, resp)
	} else {
		return rumerrors.NewBadRequestError(fmt.Sprintf("Group %s not exist", groupid))
	}
}

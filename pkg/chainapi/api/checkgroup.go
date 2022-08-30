package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func (h *Handler) CheckGroup(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	logger := c.Logger()
	params := new(handlers.LeaveGroupParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res := "ok"
	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError("group not exist")
	}

	blocknum := group.Item.HighestHeight
	genesisBlockId := group.Item.GenesisBlock.BlockId
	nodename := group.ChainCtx.GetNodeName()
	logger.Debugf("group Highest Blk id: %s HighestHeight: %d genesis Blk id: %s", group.Item.HighestBlockId, blocknum, genesisBlockId)
	blk, err := group.GetBlock(group.Item.HighestBlockId)
	prevblkid := blk.PrevBlockId
	totalblock := 1
	errblock := 0
	for {
		blocknum--
		if prevblkid == "" {
			logger.Debug("reach to the genesus:", blk.BlockId)
			break
		}
		blk, err = group.GetBlock(prevblkid)
		if err != nil {
			logger.Debugf("try to get blkid %s", prevblkid)
			logger.Debug(err)
			break
		}
		totalblock++
		prevblkid = blk.PrevBlockId

		logger.Debugf("check blkid: %s prev blkid: %s", blk.BlockId, prevblkid)
		subBlocks, err2 := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(blk.BlockId, nodename)
		if err2 != nil {
			logger.Debugf("try to get subblock err blkid %s", blk.BlockId)
			logger.Debug(err2)
			break
		}
		if len(subBlocks) == 0 {
			logger.Debugf("can't find subblocks. blockid timestamp %d blocknum %d quit", blk.TimeStamp, blocknum)
			errblock++
			//break
		}
	}
	logger.Debugf("total block: %d err block %d", totalblock, errblock)

	return c.JSON(http.StatusOK, res)
}

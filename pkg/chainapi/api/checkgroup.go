package api

import (
	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"log"
	"net/http"
)

func (h *Handler) CheckGroup(c echo.Context) (err error) {
	params := new(handlers.LeaveGroupParam)

	if err := c.Bind(params); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	res := "ok"
	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if ok {
		log.Println("======== check group:", params.GroupId, " start ========")
		blocknum := group.Item.HighestHeight
		genesisBlockId := group.Item.GenesisBlock.BlockId
		nodename := group.ChainCtx.GetNodeName()
		log.Printf("group Highest Blk id: %s HighestHeight: %d genesis Blk id: %s", group.Item.HighestBlockId, blocknum, genesisBlockId)
		blk, err := group.GetBlock(group.Item.HighestBlockId)
		prevblkid := blk.PrevBlockId
		totalblock := 1
		errblock := 0
		for {
			blocknum--
			if prevblkid == "" {
				log.Println("reach to the genesus:", blk.BlockId)
				break
			}
			blk, err = group.GetBlock(prevblkid)
			if err != nil {
				log.Printf("try to get blkid %s", prevblkid)
				log.Println(err)
				break
			}
			totalblock++
			prevblkid = blk.PrevBlockId

			log.Printf("check blkid: %s prev blkid: %s", blk.BlockId, prevblkid)
			subBlocks, err2 := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(blk.BlockId, nodename)
			if err2 != nil {
				log.Printf("try to get subblock err blkid %s", blk.BlockId)
				log.Println(err2)
				break
			}
			if len(subBlocks) == 0 {
				log.Printf("can't find subblocks. blockid timestamp %d blocknum %d quit", blk.TimeStamp, blocknum)
				errblock++
				//break
			}
		}
		log.Printf("total block: %d err block %d", totalblock, errblock)

	} else {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "group not exist"})
	}
	return c.JSON(http.StatusOK, res)
}

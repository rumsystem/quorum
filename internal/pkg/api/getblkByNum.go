package api

import (
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v3"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

func (h *Handler) GetBlockByNum(c echo.Context) (err error) {
	output := make(map[string]string)
	blocknum := c.Param("block_num")
	if blocknum == "" {
		output[ERROR_INFO] = "block_num can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	bn, err := strconv.ParseInt(blocknum, 10, 64)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	blockId, err := chain.GetDbMgr().GetBlkId(bn, groupid)
	if err != nil {

		output[ERROR_INFO] = err.Error()
		if err == badger.ErrKeyNotFound {
			return c.JSON(http.StatusNotFound, output)
		}
		return c.JSON(http.StatusBadRequest, output)
	}

	block, err := chain.GetDbMgr().GetBlock(blockId)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, block)
}

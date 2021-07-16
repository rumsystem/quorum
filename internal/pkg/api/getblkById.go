package api

import (
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

func (h *Handler) GetBlockById(c echo.Context) (err error) {
	output := make(map[string]string)
	blockid := c.Param("block_id")
	if blockid == "" {
		output[ERROR_INFO] = "block_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	block, err := chain.GetDbMgr().GetBlock(blockid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, block)
}

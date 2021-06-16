package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type GetBlockParam struct {
	BlockId string `from:"block_id" json:"block_id" validate:"required"`
}

func (h *Handler) GetBlock(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(GetBlockParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	block, err := chain.GetDbMgr().GetBlock(params.BlockId)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, block)
}

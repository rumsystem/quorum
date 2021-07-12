package api

import (
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type GetTrxParam struct {
	TrxId string `from:"trx_id" json:"trx_id" validate:"required"`
}

func (h *Handler) GetTrx(c echo.Context) (err error) {
	output := make(map[string]string)

	trxid := c.Param("trx_id")

	if trxid == "" {
		output[ERROR_INFO] = "trx_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	trx, err := chain.GetDbMgr().GetTrx(trxid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, trx)
}

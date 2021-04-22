package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type GetTrxParam struct {
	TrxId string `from:"trx_id" json:"trx_id" validate:"required"`
}

func (h *Handler) GetTrx(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(GetTrxParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	trx, err := chain.GetDbMgr().GetTrx(params.TrxId)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, trx)
}

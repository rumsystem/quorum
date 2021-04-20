package api

import (
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
)

type PostToGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
	Content string `from:"content"  json:"content"  validate:"required"`
}

func (h *Handler) PostToGroup(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(PostToGroupParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetChainCtx().Groups[params.GroupId]; ok {
		trxId, err := group.Post(params.Content)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		output[TRX_ID] = trxId
		return c.JSON(http.StatusOK, output)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}
}

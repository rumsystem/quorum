package api

import (
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type GetGroupCtnParams struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

func (h *Handler) GetGroupCtn(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(GetGroupCtnParams)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetChainCtx().Groups[params.GroupId]; ok {
		ctnList, err := chain.GetDbMgr().GetGrpCtnt(group.Item.GroupId)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		return c.JSON(http.StatusOK, ctnList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}
}

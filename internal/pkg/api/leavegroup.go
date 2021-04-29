package api

import (
	"fmt"
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type LeaveGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

func (h *Handler) LeaveGroup(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(LeaveGroupParam)

	if err := c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetChainCtx().Groups[params.GroupId]; ok {
		err := group.LeaveGrp()

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		delete(chain.GetChainCtx().Groups, params.GroupId)

		output[GROUP_ID] = params.GroupId
		output[SIGNATURE] = "Owner Signature"
		return c.JSON(http.StatusOK, output)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}
}

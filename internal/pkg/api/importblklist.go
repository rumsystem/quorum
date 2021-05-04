package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/golang/protobuf/proto"
	"github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
)

type ImportBlkListParam struct {
	Items []string
}

func (h *Handler) ImportBlkList(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(ImportBlkListParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	for _, blkItem := range params.Items {

		var item *quorumpb.BlockListItem
		item = &quorumpb.BlockListItem{}

		err := proto.Unmarshal([]byte(blkItem), item)

		err = chain.GetDbMgr().AddBlkList(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
	}

	return c.JSON(http.StatusOK, output)
}

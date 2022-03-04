package api

import (
	"github.com/go-playground/validator/v10"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

type RexSessionParam struct {
	PeerId  string `from:"peer_id"      json:"peer_id"      validate:"required,max=53,min=53"`
	GroupId string `from:"group_id"  json:"group_id"      validate:"required,max=36,min=36"`
}

func (h *Handler) RexInitSession(node *p2p.Node) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		validate := validator.New()
		params := new(RexSessionParam)
		output := make(map[string]interface{})
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		err = handlers.RexInitSession(node, params.GroupId, params.PeerId)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, params)
	}
}

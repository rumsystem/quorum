package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

func (h *Handler) RexTest(node *p2p.Node) echo.HandlerFunc {
	return func(c echo.Context) error {
		result, err := handlers.RexTest(node)
		if err != nil {
			fmt.Printf("json.Marshal failed: %s", err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

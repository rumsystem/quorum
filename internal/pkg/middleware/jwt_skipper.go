package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func JWTSkipper(c echo.Context) bool {
	if LocalhostSkipper(c) {
		return true
	}

	path := c.Request().URL.Path
	skipPathPrefix := []string{
		"/api/v1/ws/trx",
	}
	for _, v := range skipPathPrefix {
		if strings.HasPrefix(path, v) {
			return true
		}
	}

	return false
}

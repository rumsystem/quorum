package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func LocalhostSkipper(c echo.Context) bool {
	host := c.Request().Host
	skipHosts := []string{"localhost", "127.0.0.1"}
	for _, h := range skipHosts {
		if strings.HasPrefix(host, h+":") || host == h {
			return true
		}
	}

	return false
}

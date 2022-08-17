package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// ChainGzipSkipper chain api server skipper, return true: do not gzip
func ChainGzipSkipper(c echo.Context) bool {
	// skip localhost
	if LocalhostSkipper(c) {
		return true
	}

	// gzip enable for nodesdk rest api
	if !strings.HasPrefix(c.Path(), "/api/v1/node/") {
		return true
	}

	return false
}

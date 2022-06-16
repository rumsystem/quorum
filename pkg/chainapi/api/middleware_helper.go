package api

import (
	"strings"

	"github.com/labstack/echo/v4"
	appapi "github.com/rumsystem/quorum/pkg/chainapi/appapi"
)

func localhostSkipper(c echo.Context) bool {
	host := c.Request().Host
	skipHosts := []string{"localhost", "127.0.0.1"}
	for _, h := range skipHosts {
		if strings.HasPrefix(host, h+":") || host == h {
			return true
		}
	}

	return false
}

func opaInputFunc(c echo.Context) interface{} {
	r := c.Request()
	return map[string]interface{}{
		"method":       r.Method,
		"path":         strings.Split(strings.Trim(r.URL.Path, "/"), "/"),
		"role":         appapi.GetJWTRole(c),
		"allow_groups": appapi.GetJWTAllowGroups(c),
	}
}

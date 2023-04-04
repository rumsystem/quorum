package api

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	appapi "github.com/rumsystem/quorum/pkg/chainapi/appapi"
)

var opaLogger = logging.Logger("opa")

func opaInputFunc(c echo.Context) interface{} {
	token, err := appapi.GetJWTToken(c)
	if err != nil {
		opaLogger.Warnf("get jwt failed: %s", err)
		return nil
	}

	r := c.Request()
	return map[string]interface{}{
		"method":       r.Method,
		"path":         strings.Split(strings.Trim(r.URL.Path, "/"), "/"),
		"role":         appapi.GetJWTRole(token),
		"allow_groups": appapi.GetJWTAllowGroups(token),
	}
}

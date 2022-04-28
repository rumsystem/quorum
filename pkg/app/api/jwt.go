package api

import (
	"errors"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

var (
	logger = logging.Logger("api")
)

const (
	jwtContextKey = "token"
)

type TokenItem struct {
	Token string `json:"token"`
}

func getJWTKey(h *Handler) (string, error) {
	// get JWTKey from node options config file
	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return "", errors.New("Call InitNodeOptions() before use it")
	}
	return nodeOpt.JWTKey, nil
}

func getToken(name, role, jwtKey string) (string, error) {
	// FIXME: hardcode
	exp := time.Now().Add(time.Hour * 24 * 30)
	return utils.NewJWTToken(name, role, jwtKey, exp)
}

func jwtFromHeader(c echo.Context, header string, authScheme string) (string, error) {
	parts := strings.Split(header, ":")
	auth := c.Request().Header.Get(parts[1])
	l := len(authScheme)
	if len(auth) > l+1 && auth[:l] == authScheme {
		return auth[l+1:], nil
	}
	return "", errors.New("missing jwt token")
}

func CustomJWTConfig(jwtKey string) middleware.JWTConfig {
	config := middleware.JWTConfig{
		SigningMethod: "HS256",
		SigningKey:    []byte(jwtKey),
		AuthScheme:    "Bearer",
		TokenLookup:   "header:" + echo.HeaderAuthorization,
		ContextKey:    jwtContextKey,
		Skipper: func(c echo.Context) bool {
			r := c.Request()
			if strings.HasPrefix(r.Host, "localhost:") || r.Host == "localhost" || strings.HasPrefix(r.Host, "127.0.0.1") {
				return true
			} else if strings.HasPrefix(r.URL.Path, "/app/api/v1/token/apply") {
				// FIXME: hardcode url path
				return true
			}

			return false
		},
	}

	return config
}

func GetJWTRole(c echo.Context, jwtContextKey string) string {
	token := c.Get(jwtContextKey).(*jwt.Token)
	if token == nil {
		return ""
	}

	claims := token.Claims.(jwt.MapClaims)
	role, ok := claims["role"]
	if !ok {
		return ""
	}

	return role.(string)
}

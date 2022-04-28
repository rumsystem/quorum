package api

import (
	"errors"
	"net/http"
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

// @Tags Apps
// @Summary RefreshToken
// @Description Get a new auth token
// @Produce json
// @Param Authorization header string true "current auth token"
// @Success 200 {object} TokenItem  "a new auth token"
// @Router /app/api/v1/token/refresh [post]
func (h *Handler) RefreshToken(c echo.Context) error {
	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Call InitNodeOptions() before use it",
		})
	}

	// check token
	jwtKey, err := getJWTKey(h)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	// token invalid include expired or invalid
	token := c.Get(jwtContextKey).(*jwt.Token)
	tokenStr := token.Raw
	if utils.IsJWTTokenExpired(tokenStr, jwtKey) {
		logger.Infof("token expires, return new token")
	} else if valid, err := utils.IsJWTTokenValid(tokenStr, jwtKey); !valid || err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	role := GetJWTRole(c, jwtContextKey)
	newTokenStr, err := getToken(h.PeerName, role, jwtKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}
	if err := nodeOpt.SetJWTToken(role, newTokenStr); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: newTokenStr})
}

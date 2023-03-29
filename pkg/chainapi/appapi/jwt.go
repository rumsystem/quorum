package appapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	rummiddleware "github.com/rumsystem/quorum/internal/pkg/middleware"
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
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI1MTNiZDNmMi1hMGJjLTQ3MGItODA2My1lYzk1NDlmMzRiN2QiXSwiZXhwIjoxODI2MTA4MzA5LCJuYW1lIjoiYWxsb3ctNTEzYmQzZjItYTBiYy00NzBiLTgwNjMtZWM5NTQ5ZjM0YjdkIiwicm9sZSI6Im5vZGUifQ.CZQB2jzvY3lB_XgAd8izAQaunsHZFh1qN0tmSdYkce8"`
}

type CreateJWTParams struct {
	Name        string    `json:"name" validate:"required" example:"allow-513bd3f2-a0bc-470b-8063-ec9549f34b7d"`
	Role        string    `json:"role" validate:"required,oneof=node chain" example:"node"`
	AllowGroups []string  `json:"allow_groups" example:"513bd3f2-a0bc-470b-8063-ec9549f34b7d"`
	ExpiresAt   time.Time `json:"expires_at" validate:"required" example:"2022-12-28T08:10:36.675204+00:00"`
}

func getJWTKey() (string, error) {
	// get JWTKey from node options config file
	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return "", errors.New("Call InitNodeOptions() before use it")
	}
	return nodeOpt.JWTKey, nil
}

func getToken(name string, role string, allowGroups []string, jwtKey string) (string, error) {
	// FIXME: hardcode
	exp := time.Now().Add(time.Hour * 24 * 30)
	return utils.NewJWTToken(name, role, allowGroups, jwtKey, exp)
}

func CustomJWTConfig(jwtKey string) middleware.JWTConfig {
	config := middleware.JWTConfig{
		SigningMethod: "HS256",
		SigningKey:    []byte(jwtKey),
		AuthScheme:    "Bearer",
		TokenLookup:   "header:" + echo.HeaderAuthorization,
		ContextKey:    jwtContextKey,
		ParseTokenFunc: func(auth string, c echo.Context) (interface{}, error) {
			return utils.ParseJWTToken(auth, jwtKey)
		},
		Skipper: rummiddleware.JWTSkipper,
	}

	return config
}

func GetJWTName(c echo.Context) string {
	token, err := getJWTToken(c)
	if err != nil {
		logger.Errorf("get jwt token failed: %s", err)
		return ""
	}
	claims := token.Claims.(jwt.MapClaims)
	name, ok := claims["name"]
	if !ok {
		return ""
	}

	return name.(string)
}

func GetJWTRole(c echo.Context) string {
	token, err := getJWTToken(c)
	if err != nil {
		logger.Errorf("get jwt token failed: %s", err)
		return ""
	}
	claims := token.Claims.(jwt.MapClaims)
	role, ok := claims["role"]
	if !ok {
		return ""
	}

	return role.(string)
}

func GetJWTAllowGroups(c echo.Context) []string {
	groups := []string{}
	token, err := getJWTToken(c)
	if err != nil {
		logger.Errorf("get jwt token failed: %s", err)
		return groups
	}
	claims := token.Claims.(jwt.MapClaims)
	allowGroups, ok := claims["allowGroups"]
	if !ok {
		return groups
	}

	items, ok := allowGroups.([]interface{})
	if !ok {
		logger.Errorf("cast allowGroups to `[]interface{}` failed")
		return groups
	}

	for _, v := range items {
		groups = append(groups, v.(string))
	}
	return groups
}

// @Tags Apps
// @Summary CreateToken
// @Description Create a new auth token, only allow access from localhost
// @Accept  json
// @Produce json
// @Param   create_jwt_params  body CreateJWTParams  true  "create jwt params"
// @Success 200 {object} TokenItem  "a new auth token"
// @Router /app/api/v1/token/create [post]
func (h *Handler) CreateToken(c echo.Context) error {
	cc := c.(*utils.CustomContext)

	var err error

	if !rummiddleware.LocalhostSkipper(c) {
		return rumerrors.NewBadRequestError("only localhost can access this rest api")
	}

	params := new(CreateJWTParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	if params.Role == "node" {
		if params.AllowGroups == nil || len(params.AllowGroups) == 0 {
			return rumerrors.NewBadRequestError("allow_groups field must not be empty for node jwt")
		}
	} else {
		if params.AllowGroups != nil || len(params.AllowGroups) > 0 {
			return rumerrors.NewBadRequestError("allow_groups field must be empty for chain jwt")
		}
	}

	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return errors.New("Call InitNodeOptions() before use it")
	}

	jwtKey, err := getJWTKey()
	if err != nil {
		return err
	}

	tokenStr, err := utils.NewJWTToken(params.Name, params.Role, params.AllowGroups, jwtKey, params.ExpiresAt)
	if err != nil {
		return err
	}
	if err := nodeOpt.SetJWTTokenMap(params.Name, tokenStr); err != nil {
		return errors.New("save jwt to config file failed")
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: tokenStr})
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
		return errors.New("Call InitNodeOptions() before use it")
	}

	// check token
	jwtKey, err := getJWTKey()
	if err != nil {
		return err
	}

	token, err := getJWTToken(c)
	if err != nil {
		return err
	}

	// token invalid include expired or invalid
	tokenStr := token.Raw
	if utils.IsJWTTokenExpired(tokenStr, jwtKey) {
		logger.Infof("token expires, return new token")
	} else if valid, err := utils.IsJWTTokenValid(tokenStr, jwtKey); !valid || err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	name := GetJWTName(c)
	role := GetJWTRole(c)
	allowGroups := GetJWTAllowGroups(c)

	newTokenStr, err := getToken(name, role, allowGroups, jwtKey)
	if err != nil {
		return err
	}
	if err := nodeOpt.SetJWTTokenMap(name, newTokenStr); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: newTokenStr})
}

func jwtFromHeader(c echo.Context) (string, error) {
	config := CustomJWTConfig("")
	header := config.TokenLookup
	authScheme := config.AuthScheme
	parts := strings.Split(header, ":")
	auth := c.Request().Header.Get(parts[1])
	l := len(authScheme)
	if len(auth) > l+1 && auth[:l] == authScheme {
		return auth[l+1:], nil
	}
	return "", errors.New("missing jwt token")
}

// getJWTToken get jwt token from echo context or http request header
// can not get jwt token from c.Get(jwtContextKey) for localhost or 127.0.0.1
func getJWTToken(c echo.Context) (*jwt.Token, error) {
	token := c.Get(jwtContextKey)
	if token != nil {
		return (token.(*jwt.Token)), nil
	}

	tokenStr, err := jwtFromHeader(c)
	if err != nil {
		return nil, err
	}

	jwtKey, err := getJWTKey()
	if err != nil {
		return nil, errors.New("can not get jwt key")
	}
	return utils.ParseJWTToken(tokenStr, jwtKey)
}

package appapi

import (
	"errors"
	"fmt"
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
	Name      string    `json:"name" validate:"required" example:"allow-513bd3f2-a0bc-470b-8063-ec9549f34b7d"`
	Role      string    `json:"role" validate:"required,oneof=node chain" example:"node"`
	GroupId   string    `json:"group_id" validate:"required_if=Role node" example:"513bd3f2-a0bc-470b-8063-ec9549f34b7d"`
	ExpiresAt time.Time `json:"expires_at" validate:"required" example:"2022-12-28T08:10:36.675204+00:00"`
}

type RevokeJWTParams struct {
	Role    string `json:"role" validate:"required,oneof=node chain" example:"node"`
	GroupId string `json:"group_id" validate:"required_if=Role node" example:"513bd3f2-a0bc-470b-8063-ec9549f34b7d"`
	Token   string `json:"token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI1MTNiZDNmMi1hMGJjLTQ3MGItODA2My1lYzk1NDlmMzRiN2QiXSwiZXhwIjoxODI2MTA4MzA5LCJuYW1lIjoiYWxsb3ctNTEzYmQzZjItYTBiYy00NzBiLTgwNjMtZWM5NTQ5ZjM0YjdkIiwicm9sZSI6Im5vZGUifQ.CZQB2jzvY3lB_XgAd8izAQaunsHZFh1qN0tmSdYkce8"`
}

type RemoveJWTParams struct {
	Role    string `json:"role" query:"role" validate:"required,oneof=node chain" example:"node"`
	GroupId string `json:"group_id" query:"group_id" validate:"required_if=Role node" example:"513bd3f2-a0bc-470b-8063-ec9549f34b7d"`
	Token   string `json:"token" query:"token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI1MTNiZDNmMi1hMGJjLTQ3MGItODA2My1lYzk1NDlmMzRiN2QiXSwiZXhwIjoxODI2MTA4MzA5LCJuYW1lIjoiYWxsb3ctNTEzYmQzZjItYTBiYy00NzBiLTgwNjMtZWM5NTQ5ZjM0YjdkIiwicm9sZSI6Im5vZGUifQ.CZQB2jzvY3lB_XgAd8izAQaunsHZFh1qN0tmSdYkce8"`
}

func getJWTKey() (string, error) {
	// get JWTKey from node options config file
	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return "", errors.New("Call InitNodeOptions() before use it")
	}
	return nodeOpt.JWT.Key, nil
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

func GetJWTName(token *jwt.Token) string {
	claims := token.Claims.(jwt.MapClaims)
	name, ok := claims["name"]
	if !ok {
		return ""
	}

	return name.(string)
}

func GetJWTRole(token *jwt.Token) string {
	claims := token.Claims.(jwt.MapClaims)
	role, ok := claims["role"]
	if !ok {
		return ""
	}

	return role.(string)
}

func GetJWTAllowGroups(token *jwt.Token) []string {
	groups := []string{}
	claims := token.Claims.(jwt.MapClaims)
	allowGroup, ok := claims["allowGroup"]
	if !ok {
		return groups
	}

	item, ok := allowGroup.(string)
	if !ok {
		logger.Errorf("cast allowGroup to `string` failed")
		return groups
	}

	return []string{item}
}

// @Tags Apps
// @Summary CreateToken
// @Description Create a new auth token, only allow access from localhost
// @Accept  json
// @Produce json
// @Param   create_jwt_params  body CreateJWTParams  true  "create jwt params"
// @Success 200 {object} TokenItem  "a new auth token"
// @Router /app/api/v1/token [post]
func (h *Handler) CreateToken(c echo.Context) error {
	var err error
	cc := c.(*utils.CustomContext)
	params := new(CreateJWTParams)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return errors.New("Call InitNodeOptions() before use it")
	}

	var tokenStr string
	if params.Role == "chain" {
		tokenStr, err = nodeOpt.NewChainJWT(params.Name, params.ExpiresAt)
	} else if params.Role == "node" {
		tokenStr, err = nodeOpt.NewNodeJWT(params.GroupId, params.Name, params.ExpiresAt)
	}
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: tokenStr})
}

// @Tags Apps
// @Summary RevokeToken
// @Description Revoke a auth token
// @Accept json
// @Produce json
// @Param Authorization header string true "current auth token"
// @Param   revoke_jwt_params  body RevokeJWTParams  true  "revoke jwt params"
// @Success 200 {object} utils.SuccessResponse
// @Router /app/api/v1/token/revoke [post]
func (h *Handler) RevokeToken(c echo.Context) error {
	cc := c.(*utils.CustomContext)
	var payload RevokeJWTParams
	if err := cc.BindAndValidate(&payload); err != nil {
		return err
	}

	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return errors.New("Call InitNodeOptions() before use it")
	}

	if payload.Role == "node" {
		if err := nodeOpt.RevokeNodeJWT(payload.GroupId, payload.Token); err != nil {
			return err
		}
	} else if payload.Role == "chain" {
		if err := nodeOpt.RevokeChainJWT(payload.Token); err != nil {
			return err
		}
	}

	return cc.Success()
}

// @Tags Apps
// @Summary RemoveToken
// @Description Remove a auth token
// @Produce json
// @Param Authorization header string true "current auth token"
// @Param   remove_jwt_params  query RemoveJWTParams  true  "remove jwt params"
// @Success 200 {object} utils.SuccessResponse
// @Router /app/api/v1/token [delete]
func (h *Handler) RemoveToken(c echo.Context) error {
	cc := c.(*utils.CustomContext)
	var payload RemoveJWTParams
	if err := cc.BindAndValidate(&payload); err != nil {
		return err
	}

	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return errors.New("Call InitNodeOptions() before use it")
	}

	if payload.Role == "node" {
		if err := nodeOpt.RemoveNodeJWT(payload.GroupId, payload.Token); err != nil {
			return err
		}
	} else if payload.Role == "chain" {
		if err := nodeOpt.RemoveChainJWT(payload.Token); err != nil {
			return err
		}
	}

	return cc.Success()
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

	token, err := GetJWTToken(c)
	if err != nil {
		return rumerrors.NewBadRequestError(errors.New("invalid jwt"))
	}
	name := GetJWTName(token)
	role := GetJWTRole(token)
	allowGroups := GetJWTAllowGroups(token)
	exp := time.Now().Add(time.Hour * 24 * 30)
	var newTokenStr string

	if role == "chain" {
		if !nodeOpt.IsValidChainJWT(token.Raw) {
			return rumerrors.NewBadRequestError(errors.New("invalid token"))
		}
		newTokenStr, err = nodeOpt.NewChainJWT(name, exp)
		if err != nil {
			return err
		}
	}

	if role == "node" {
		if !nodeOpt.IsValidNodeJWT(allowGroups[0], token.Raw) {
			return rumerrors.NewBadRequestError(errors.New("invalid token"))
		}
		newTokenStr, err = nodeOpt.NewNodeJWT(allowGroups[0], name, exp)
		if err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: newTokenStr})
}

// @Tags Apps
// @Summary ListToken
// @Description List all auth token
// @Produce json
// @Param Authorization header string true "current auth token"
// @Success 200 {object} options.JWT
// @Router /app/api/v1/token/list [get]
func (h *Handler) ListToken(c echo.Context) error {
	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return errors.New("Call InitNodeOptions() before use it")
	}

	tokens, err := nodeOpt.GetAllJWT()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, tokens)
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

// GetJWTToken get jwt token from echo context or http request header
// can not get jwt token from c.Get(jwtContextKey) for localhost or 127.0.0.1
func GetJWTToken(c echo.Context) (*jwt.Token, error) {
	token := c.Get(jwtContextKey)
	if token != nil {
		t := (token.(*jwt.Token))
		if _isValidToken(t) {
			return t, nil
		}
		return nil, fmt.Errorf("invalid jwt: %s", t.Raw)
	}

	tokenStr, err := jwtFromHeader(c)
	if err != nil {
		logger.Warn("can not get jwt from header")
		return nil, err
	}

	jwtKey, err := getJWTKey()
	if err != nil {
		e := errors.New("can not get jwt key")
		logger.Warn(e)
		return nil, e
	}

	_token, err := utils.ParseJWTToken(tokenStr, jwtKey)
	if err != nil {
		e := fmt.Errorf("parse jwt token failed: %s", err)
		logger.Warn(e)
		return nil, e
	}

	if _isValidToken(_token) {
		return _token, nil
	}

	e := fmt.Errorf("invalid jwt: %s", _token.Raw)
	logger.Warn(e)
	return nil, e
}

func _isValidToken(token *jwt.Token) bool {
	role := GetJWTRole(token)
	groupid := GetJWTAllowGroups(token)[0]

	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		logger.Warn("Call InitNodeOptions() before use it")
		return false
	}

	if role == "chain" {
		if nodeOpt.IsValidChainJWT(token.Raw) {
			return true
		} else {
			logger.Warn("invalid chain jwt")
			return false
		}
	} else if role == "node" {
		if nodeOpt.IsValidNodeJWT(groupid, token.Raw) {
			return true
		} else {
			logger.Warn(errors.New("invalid node jwt"))
			return false
		}
	}

	logger.Warn(errors.New("invalid jwt role"))
	return false
}

package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/huo-ju/quorum/internal/pkg/options"
	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	logger = logging.Logger("api")
)

func getJWTKey(h *Handler) (string, error) {
	// get JWTKey from node options config file
	nodeOpt, err := options.Load(h.ConfigDir, h.PeerName)
	if err != nil {
		return "", err
	}

	return nodeOpt.JWTKey, nil
}

func getToken(name string, jwtKey string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = name
	// FIXME: hardcode
	claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix()

	return token.SignedString([]byte(jwtKey))
}

func (h *Handler) ApplyToken(c echo.Context) error {
	nodeOpt, err := options.Load(h.ConfigDir, h.PeerName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}
	if nodeOpt.JWTToken != "" {
		// already generate jwt token; return 400
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "please find jwt token in peer options; if want to refresh token, access /token/refresh",
		})
	}

	jwtKey, err := getJWTKey(h)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	tokenStr, err := getToken(h.PeerName, jwtKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	if err := options.SetJWTToken(h.ConfigDir, h.PeerName, tokenStr); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"token": tokenStr,
	})
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

func (h *Handler) RefreshToken(c echo.Context) error {
	config := CustomJWTConfig("")
	tokenStr, err := jwtFromHeader(c, config.TokenLookup, config.AuthScheme)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	// check token
	jwtKey, err := getJWTKey(h)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})

	if err != nil {
		e := err.(*jwt.ValidationError)
		if e.Errors == jwt.ValidationErrorExpired {
			logger.Infof("token expires, return new token")
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
		}
	}

	newTokenStr, err := getToken(h.PeerName, jwtKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	if err := options.SetJWTToken(h.ConfigDir, h.PeerName, newTokenStr); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"token": newTokenStr,
	})
}

func CustomJWTConfig(jwtKey string) middleware.JWTConfig {
	config := middleware.JWTConfig{
		SigningMethod: "HS256",
		SigningKey:    []byte(jwtKey),
		AuthScheme:    "Bearer",
		TokenLookup:   "header:" + echo.HeaderAuthorization,
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

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

func getToken(name string, jwtKey string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = name
	// FIXME: hardcode
	claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix()

	return token.SignedString([]byte(jwtKey))
}

//https://localhost:8002/app/api/v1/token/apply
//curl -k -X POST -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzU1NDk1NTAsIm5hbWUiOiJwZWVyMiJ9.zMbTmoIEZhyjVtHpIF5Uy5cJClDVR1pB6W_DsrC9GcA"  https://localhost:8002/app/api/v1/token/refresh

//{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzU1NDk2NDksIm5hbWUiOiJwZWVyMiJ9.ZXJBY0s_SqRcCM7_eM2LCQcjsZwY1epTby19O8lf_dk"}

// @Tags Apps
// @Summary GetAuthToken
// @Description Get a auth token for authorizing requests from remote
// @Produce json
// @Param Authorization header string false "current auth token"
// @Success 200 {object} TokenItem  "a auth token"
// @Router /app/api/v1/token/apply [post]
func (h *Handler) ApplyToken(c echo.Context) error {
	nodeOpt := options.GetNodeOptions()
	if nodeOpt == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Call InitNodeOptions() before use it",
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

	if err := nodeOpt.SetJWTToken(tokenStr); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: tokenStr})
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

// @Tags Apps
// @Summary RefreshToken
// @Description Get a new auth token
// @Produce json
// @Param Authorization header string true "current auth token"
// @Success 200 {object} TokenItem  "a new auth token"
// @Router /app/api/v1/token/refresh [post]
func (h *Handler) RefreshToken(c echo.Context) error {
	config := CustomJWTConfig("")
	tokenStr, err := jwtFromHeader(c, config.TokenLookup, config.AuthScheme)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

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

	if err := nodeOpt.SetJWTToken(newTokenStr); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &TokenItem{Token: newTokenStr})
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

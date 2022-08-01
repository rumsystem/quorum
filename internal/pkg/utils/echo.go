package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

const sep = ","

type (
	CustomContext struct {
		echo.Context
	}
	CustomValidator struct {
		validator *validator.Validate
		trans     *ut.Translator
	}
	ValidationWithTransError struct {
		errs  validator.ValidationErrors
		trans ut.Translator
	}
	CustomBinder struct{}
)

func NewEcho(debug bool) *echo.Echo {
	e := echo.New()

	// enable or disable debug mode
	e.Debug = debug

	// config log level and format
	// ref: https://echo.labstack.com/guide/customization/#logging
	if debug {
		e.Logger.SetLevel(log.DEBUG)
	} else {
		e.Logger.SetLevel(log.INFO)
	}
	e.Logger.SetHeader("${time_rfc3339_nano} ${level} ${prefix} ${short_file} ${line}")

	// hide banner
	e.HideBanner = true

	e.Binder = new(CustomBinder)
	e.Validator = NewCustomValidator()
	e.HTTPErrorHandler = customHTTPErrorHandler

	// timeout
	e.Server.ReadTimeout = 30 * time.Second
	e.Server.WriteTimeout = 30 * time.Second

	// middleware
	// this middleware should be registered before any other middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &CustomContext{Context: c}
			return next(cc)
		}
	})

	// logs the information about each HTTP request
	// ref: https://echo.labstack.com/middleware/logger/
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339_nano} method=${method} uri=${uri} status=${status} latency=${latency_human} error={${error}}\n",
	}))

	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
			http.MethodPost, http.MethodDelete, http.MethodOptions,
		},
	}))

	return e
}

func (cb *CustomBinder) Bind(i interface{}, c echo.Context) (err error) {
	db := new(echo.DefaultBinder)
	switch i.(type) {
	case *quorumpb.Activity:
		bodyBytes, err := ioutil.ReadAll(c.Request().Body)
		err = protojson.Unmarshal(bodyBytes, i.(*quorumpb.Activity))
		return err
	default:
		if err = db.Bind(i, c); err != echo.ErrUnsupportedMediaType {
			return
		}
		return err
	}
}

func (vet ValidationWithTransError) Error() string {
	buff := bytes.NewBufferString("")
	for _, v := range vet.errs.Translate(vet.trans) {
		buff.WriteString(v)
		buff.WriteString(sep)
	}

	return strings.TrimSuffix(buff.String(), sep)
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		errs := err.(validator.ValidationErrors)
		verr := ValidationWithTransError{
			errs:  errs,
			trans: *cv.trans,
		}
		return rumerrors.NewBadRequestError(verr.Error())
	}
	return nil
}

func (c *CustomContext) BindAndValidate(params interface{}) error {
	if err := c.Bind(params); err != nil {
		return err
	}
	if err := c.Validate(params); err != nil {
		return err
	}

	return nil
}

func (c *CustomContext) GetBaseURLFromRequest() string {
	scheme := "http"
	if c.Context.IsTLS() {
		scheme = "https"
	}

	host := c.Context.Request().Host

	_url := fmt.Sprintf("%s://%s", scheme, host)
	return _url
}

type ErrorResponse echo.HTTPError

type SuccessResponse struct {
	Success bool `json:"success"`
}

func (c *CustomContext) Success() error {
	res := SuccessResponse{Success: true}
	return c.JSON(http.StatusOK, res)
}

const jwtQsKey = "jwt"

func GetChainapiURL(baseUrl, jwt string) (string, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if len(q) == 0 {
		q = url.Values{}
	}
	q.Add(jwtQsKey, jwt)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ParseChainapiURL return url and jwt
func ParseChainapiURL(_url string) (baseUrl string, jwt string, err error) {
	u, err := url.Parse(_url)
	if err != nil {
		return "", "", err
	}
	q := u.Query()
	jwt = q.Get(jwtQsKey)
	q.Del(jwtQsKey)
	u.RawQuery = q.Encode()
	baseUrl = u.String()
	return baseUrl, jwt, nil
}

func NewCustomValidator() *CustomValidator {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		// struct Filed Name first letter lowercase if not specified`json:xx`
		if name == "-" || name == "" {
			return LowerFirstLetter(fld.Name)
		}
		return name
	})
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")
	if err := entranslations.RegisterDefaultTranslations(validate, trans); err != nil {
		panic(err)
	}
	return &CustomValidator{validator: validate, trans: &trans}
}

func customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	he, ok := err.(*echo.HTTPError)
	if ok {
		if he.Internal != nil {
			if herr, ok := he.Internal.(*echo.HTTPError); ok {
				he = herr
			}
		}
	} else {
		var msg string
		if err != nil {
			msg = err.Error()
		} else {
			msg = http.StatusText(http.StatusInternalServerError)
		}

		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: msg,
		}
	}

	he.Message = getErrorMessage(he, c)
	// send response
	if c.Request().Method == http.MethodHead { // Issue #608
		err = c.NoContent(he.Code)
	} else {
		err = c.JSON(he.Code, he)
	}
	if err != nil {
		c.Logger().Error(err)
	}
}

func getErrorMessage(he *echo.HTTPError, c echo.Context) string {
	if m, ok := he.Message.(error); ok {
		return m.Error()
	} else if m, ok := he.Message.(string); ok {
		return m
	}

	msg, err := json.Marshal(he.Message)
	if err != nil {
		c.Logger().Error(err)
		return http.StatusText(he.Code)
	}
	return string(msg)
}

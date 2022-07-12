package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/open-policy-agent/opa/rego"
)

type (
	// OpaConfig defines the config for opa middleware.
	OpaConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper middleware.Skipper

		// BeforeFunc defines a function which is executed just before the middleware.
		BeforeFunc middleware.BeforeFunc

		// SuccessHandler defines a function which is executed for a valid request.
		SuccessHandler OpaSuccessHandler

		// ErrorHandler defines a function which is executed for an invalid request.
		// It may be used to define a custom opa error.
		ErrorHandler OpaErrorHandler

		// ErrorHandlerWithContext is almost identical to ErrorHandler, but it's passed the current context
		ErrorHandlerWithContext OpaErrorHandlerWithContext

		// Query, rego query, e.g.: "x = data.httpapi.authz.allow"
		Query string

		// Policy, rego policy
		Policy string

		// InputFunc, generate opa rego input
		InputFunc OpaRegoInputFunc

		// EvalFunc, rego eval function
		EvalFunc OpaRegoEvalFunc
	}

	// OpaSuccessHandler defines a function which is executed for a valid token.
	OpaSuccessHandler func(echo.Context)

	// OpaErrorHandler defines a function which is executed for an invalid token.
	OpaErrorHandler func(error) error

	// OpaErrorHandlerWithContext is almost identical to JWTErrorHandler, but it's passed the current context.
	OpaErrorHandlerWithContext func(error, echo.Context) error

	// OpaRegoInputFunc
	OpaRegoInputFunc func(echo.Context) interface{}

	// RegoEvalFunc
	OpaRegoEvalFunc func(context.Context, rego.PreparedEvalQuery, interface{}) (bool, error)
)

// Errors
var (
	ErrNoPermission    = echo.NewHTTPError(http.StatusUnauthorized, "opa no permission error")
	ErrEval            = echo.NewHTTPError(http.StatusInternalServerError, "opa eval error")
	ErrUndefinedResult = echo.NewHTTPError(http.StatusInternalServerError, "opa undefined result error")
)

var (
	DefaultOpaConfig = OpaConfig{
		Skipper:  middleware.DefaultSkipper,
		Policy:   "",
		Query:    "",
		EvalFunc: nil,
	}
)

func DefaultEvalFunc(ctx context.Context, query rego.PreparedEvalQuery, input interface{}) (bool, error) {
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, ErrEval
	} else if len(results) == 0 {
		return false, ErrUndefinedResult
	} else {
		for _, item := range results {
			for _, v := range item.Bindings {
				if !v.(bool) {
					return false, ErrNoPermission
				}
			}
		}

		return true, nil
	}
}

// OpaWithConfig returns a opa auth middleware with config.
func OpaWithConfig(config OpaConfig) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = DefaultOpaConfig.Skipper
	}

	if config.Policy == "" {
		panic("echo: opa middleware requires opa rego policy")
	}

	if config.Query == "" {
		panic("echo: opa middleware requires opa rego query")
	}

	if config.InputFunc == nil {
		panic("echo: opa middleware requires opa rego input function")
	}

	if config.EvalFunc == nil {
		// panic("echo: opa middleware requires opa rego eval function")
		config.EvalFunc = DefaultEvalFunc
	}

	ctx := context.Background()

	query, err := rego.New(
		rego.Query(config.Query),
		rego.Module("policy.rego", config.Policy),
	).PrepareForEval(ctx)
	if err != nil {
		panic(fmt.Sprintf("echo: opa middleware initial opa failed: %s", err))
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			if config.BeforeFunc != nil {
				config.BeforeFunc(c)
			}

			input := config.InputFunc(c)
			ctx := context.Background()
			ok, err := config.EvalFunc(ctx, query, input)
			if err != nil {
				if config.ErrorHandler != nil {
					return config.ErrorHandler(err)
				}
				if config.ErrorHandlerWithContext != nil {
					return config.ErrorHandlerWithContext(err, c)
				}

				return err
			}

			if !ok {
				err := ErrNoPermission
				if config.ErrorHandler != nil {
					return config.ErrorHandler(err)
				}
				if config.ErrorHandlerWithContext != nil {
					return config.ErrorHandlerWithContext(err, c)
				}

				return err
			}

			if config.SuccessHandler != nil {
				config.SuccessHandler(c)
			}
			return next(c)
		}
	}
}

package api

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type CustomValidatorProfile struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorProfile) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type != handlers.Update {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unknown type of Actitity: %s, expect: %s", inputobj.Type, handlers.Update))
		}

		if inputobj.Person == nil || inputobj.Target == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Person or Target is nil"))
		}

		if inputobj.Target.Type == handlers.Group {
			if inputobj.Target.Id == "" {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Target Group must not be nil"))
			}

			if inputobj.Person.Name == "" && inputobj.Person.Image == nil && inputobj.Person.Wallet == nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Person must have name or image fields"))
			}
		}
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return rumerrors.NewInternalServerError(err)
		}
	}
	return nil
}

// @Tags User
// @Summary UpdateProfile
// @Description Update user profile
// @Accept json
// @Produce json
// @Param data body quorumpb.Activity true "Activity object"
// @Success 200 {object} handlers.UpdateProfileResult
// @Router /api/v1/group/profile [post]
func (h *Handler) UpdateProfile(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	paramspb := new(quorumpb.Activity)
	if err := cc.BindAndValidate(paramspb); err != nil {
		return err
	}

	sudo := false
	if c.Param("sudo") != "" {
		sudo, err = strconv.ParseBool(c.Param("sudo"))
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
	}

	res, err := handlers.UpdateProfile(paramspb, sudo)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

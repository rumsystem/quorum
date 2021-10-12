package api

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
)

type CustomValidatorProfile struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorProfile) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type == Update {
			if inputobj.Person != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Person.Name != "" || inputobj.Person.Image != nil || inputobj.Person.Wallet != nil {
						return nil
					}
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Person must have name or image fields"))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Target Group must not be nil"))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Object and Target Object must not be nil"))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unknown type of Actitity: %s", inputobj.Type))
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return nil
}

func (h *Handler) UpdateProfile(c echo.Context) (err error) {

	output := make(map[string]string)
	paramspb := new(quorumpb.Activity)
	if err = c.Bind(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	validate := &CustomValidatorProfile{Validator: validator.New()}
	if err = validate.Validate(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[paramspb.Target.Id]; ok {
		if paramspb.Person.Image != nil {
			_, formatname, err := image.Decode(bytes.NewReader(paramspb.Person.Image.Content))
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			if fmt.Sprintf("image/%s", formatname) != strings.ToLower(paramspb.Person.Image.MediaType) {
				output[ERROR_INFO] = fmt.Sprintf("image format don't match, mediatype is %s but the file is %s", strings.ToLower(paramspb.Person.Image.MediaType), fmt.Sprintf("image/%s", formatname))
				return c.JSON(http.StatusBadRequest, output)
			}
		}

		trxId, err := group.PostToGroup(paramspb.Person)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		output[TRX_ID] = trxId
		return c.JSON(http.StatusOK, output)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", paramspb.Target.Id)
		return c.JSON(http.StatusBadRequest, output)
	}
}

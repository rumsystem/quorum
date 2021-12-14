package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type CustomValidatorProfile struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorProfile) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type != Update {
			return errors.New(fmt.Sprintf("unknown type of Actitity: %s, expect: %s", inputobj.Type, Update))
		}

		if inputobj.Person == nil || inputobj.Target == nil {
			return errors.New(fmt.Sprintf("Person or Target is nil"))
		}

		if inputobj.Target.Type == Group {
			if inputobj.Target.Id == "" {
				return errors.New(fmt.Sprintf("Target Group must not be nil"))
			}

			if inputobj.Person.Name == "" && inputobj.Person.Image == nil && inputobj.Person.Wallet == nil {
				return errors.New(fmt.Sprintf("Person must have name or image fields"))
			}
		}
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return errors.New(err.Error())
		}
	}
	return nil
}

type UpdateProfileResult struct {
	TrxID string `json:"trx_id" validate:"required"`
}

func UpdateProfile(paramspb *quorumpb.Activity) (*UpdateProfileResult, error) {
	validate := &CustomValidatorProfile{Validator: validator.New()}
	if err := validate.Validate(paramspb); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[paramspb.Target.Id]; ok {
		if paramspb.Person.Image != nil {
			_, formatname, err := image.Decode(bytes.NewReader(paramspb.Person.Image.Content))
			if err != nil {
				return nil, err
			}
			if fmt.Sprintf("image/%s", formatname) != strings.ToLower(paramspb.Person.Image.MediaType) {
				return nil, errors.New(fmt.Sprintf("image format don't match, mediatype is %s but the file is %s", strings.ToLower(paramspb.Person.Image.MediaType), fmt.Sprintf("image/%s", formatname)))
			}
		}

		trxId, err := group.PostToGroup(paramspb.Person)

		if err != nil {
			return nil, err
		}
		result := &UpdateProfileResult{TrxID: trxId}
		return result, nil
	} else {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", paramspb.Target.Id))
	}
}

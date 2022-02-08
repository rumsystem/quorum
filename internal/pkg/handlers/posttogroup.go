package handlers

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type CustomValidatorPost struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorPost) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type == Add {
			if inputobj.Object != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Object.Type == Note && (inputobj.Object.Content != "" || len(inputobj.Object.Image) > 0) {
						return nil
					} else if inputobj.Object.Type == File && inputobj.Object.File != nil {
						return nil
					}
					return errors.New(fmt.Sprintf("unsupported object type: %s", inputobj.Object.Type))
				}
				return errors.New(fmt.Sprintf("Target Group must not be nil"))
			}
			return errors.New(fmt.Sprintf("Object and Target Object must not be nil"))
		} else if inputobj.Type == Like || inputobj.Type == Dislike {
			if inputobj.Object != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Object.Id != "" {
						return nil
					}
					return errors.New(fmt.Sprintf("unsupported object type: %s", inputobj.Object.Type))
				}
				return errors.New(fmt.Sprintf("Target Group must not be nil"))
			}
			return errors.New(fmt.Sprintf("Object and Target Object must not be nil"))
		}
		return errors.New(fmt.Sprintf("unknown type of Actitity: %s", inputobj.Type))
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return err
		}
	}
	return nil
}

type TrxResult struct {
	TrxId string `json:"trx_id" validate:"required"`
}

func PostToGroup(paramspb *quorumpb.Activity) (*TrxResult, error) {
	validate := &CustomValidatorPost{Validator: validator.New()}
	if err := validate.Validate(paramspb); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[paramspb.Target.Id]; ok {
		if paramspb.Object.Type == "" {
			paramspb.Object.Type = paramspb.Type
		}

		trxId, err := group.PostToGroup(paramspb.Object)

		if err != nil {
			return nil, err
		}
		return &TrxResult{trxId}, nil
	} else {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", paramspb.Target.Id))
	}
}

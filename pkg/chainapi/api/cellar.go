package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type AddCellearParam struct {
	GroupId    string `from:"group_id" json:"group_id" validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	CellarSeed *quorumpb.GroupSeed
	Type       string                        `from:"type" json:"type" validate:"required,oneof=brew sync" example:"brew"`
	Proof      *quorumpb.ServiceReqProofItem `from:"proof" json:"proof" validate:"required" example:"proof"`
	Memo       string                        `from:"memo" json:"memo" validate:"required" example:"memo"`
}

type AddCellearResult struct {
	GroupId string `from:"group_id" json:"group_id" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	TrxId   string `from:"trx_id" json:"trx_id" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
}

func (h *Handler) AddCellar(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(AddCellearParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	} else {
		var serviceType quorumpb.GroupServiceType
		if params.Type == "brew" {
			serviceType = quorumpb.GroupServiceType_BREW_SERVICE
		} else if params.Type == "sync" {
			serviceType = quorumpb.GroupServiceType_SYNC_SERVICE
		} else {
			return rumerrors.NewBadRequestError("Invalid service type")
		}

		trxId, err := group.ReqCellarServices(params.CellarSeed, serviceType, params.Proof, params.Memo)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, &AddCellearResult{
			GroupId: params.GroupId,
			TrxId:   trxId,
		})
	}
}

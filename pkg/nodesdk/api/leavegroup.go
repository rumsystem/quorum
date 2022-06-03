package nodesdkapi

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type LeaveGroupParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

type LeaveGroupResult struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) LeaveGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)
		validate := validator.New()

		params := new(LeaveGroupParams)

		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		//save nodesdkgroupitem to db
		dbMgr := nodesdkctx.GetDbMgr()
		err = dbMgr.RmGroup(params.GroupId)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		leaveGroupResult := &LeaveGroupResult{GroupId: params.GroupId}
		return c.JSON(http.StatusOK, leaveGroupResult)
	}
}

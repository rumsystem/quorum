package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	_ "github.com/rumsystem/rumchaindata/pkg/pb" //import for swaggo
	"google.golang.org/protobuf/encoding/protojson"
)

// @Tags Chain
// @Summary GetTrx
// @Description Get a transaction a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param trx_id path string  true "Transaction Id"
// @Success 200 {object} pb.Trx
// @Router /api/v1/trx/{group_id}/{trx_id} [get]
func (h *Handler) GetTrx(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	trxid := c.Param("trx_id")
	if trxid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidTrxID)
	}

	//should return nonce count to client?
	trx, _, err := handlers.GetTrx(groupid, trxid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	m := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	jsonString := m.Format(trx)

	return c.String(http.StatusOK, jsonString)
}

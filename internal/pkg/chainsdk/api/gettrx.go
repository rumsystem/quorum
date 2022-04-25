package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	handlers "github.com/rumsystem/quorum/internal/pkg/chainsdk/handlers"
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

	output := make(map[string]string)

	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	trxid := c.Param("trx_id")
	if trxid == "" {
		output[ERROR_INFO] = "trx_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	//should return nonce count to client?
	trx, _, err := handlers.GetTrx(groupid, trxid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	m := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	jsonString := m.Format(trx)

	return c.String(http.StatusOK, jsonString)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary GetConsensusHistory
// @Description Get the list of consensus change proof history of a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.GetConsensusHistory
// @Router /api/v1/group/{group_id}/consensus/proof/history [get]
func (h *Handler) GetConsensusHistory(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	res, err := handlers.GetConsensusHistoryHandler(h.ChainAPIdb, groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

// @Tags Management
// @Summary GetLatestConsensusChangeResult
// @Description Get the lastest change consensus result of a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.ConsensusChangeResultBundle
// @Router /api/v1/group/{group_id}/consensus/proof/last [get]
func (h *Handler) GetLatestConsensusChangeResult(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	res, err := handlers.GetLatestConsensusChangeResultHandler(h.ChainAPIdb, groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

// @Tags Management
// @Summary GetConsensusResultByReqId
// @Description Get consensus change proof by req_id
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param req_id path string  true "Req Id"
// @Success 200 {array} handlers.GetConsensusResultByReqIdResult
// @Router /api/v1/group/{group_id}/consensus/proof/{req_id} [get]
func (h *Handler) GetConsensusResultByReqId(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	reqId := c.Param("req_id")
	res, err := handlers.GetConsensusResultByReqIdHandler(h.ChainAPIdb, groupId, reqId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

// @Tags Management
// @Summary GetCurrentConsensus
// @Description Get current consensus info
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.GetCurrentConsensusResult
// @Router /api/v1/group/{group_id}/consensus/ [get]
func (h *Handler) GetCurrentConsensus(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	res, err := handlers.GetCurrentConsensusHandler(h.ChainAPIdb, groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

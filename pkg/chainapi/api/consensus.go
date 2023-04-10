package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func (h *Handler) GetConsensusHistory(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	res, err := handlers.GetConsensusHistoryHandler(h.ChainAPIdb, groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetLatestConsensusChangeResult(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	res, err := handlers.GetLatestConsensusChangeResultHandler(h.ChainAPIdb, groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetConsensusResultByReqId(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	reqId := c.Param("req_id")
	res, err := handlers.GetConsensusResultByReqIdHandler(h.ChainAPIdb, groupId, reqId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetCurrentConsensus(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	res, err := handlers.GetCurrentConsensusHandler(h.ChainAPIdb, groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func (h *Handler) GetConsensusHistory(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.GetConsensusHistoryParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.GetConsensusHistoryHandler(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetLatestConsensusChangeResult(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.GetLatestConsensusChangeResultParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.GetLatestConsensusChangeResultHandler(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetConsensusResultByReqId(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.GetConsensusResultByReqIdParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.GetConsensusResultByReqIdHandler(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetCurrentConsensus(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.GetCurrentConsensusParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.GetCurrentConsensusHandler(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}

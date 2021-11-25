package api

import (
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type RestoreParam struct {
	BackupResult
	// restore path
	Path     string `json:"path" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RestoreResult struct {
	Path string `json:"path" validate:"required"`
}

// @Tags Chain
// @Summary Restore
// @Description Restore my group seed/keystore/config from backup file
// @Produce json
// @Success 200 {object} RestoreResult
// @Router /api/v1/group/restore [post]
func (h *Handler) Restore(c echo.Context) (err error) {

	params := new(RestoreParam)
	if err := c.Bind(params); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	validate := validator.New()
	if err = validate.Struct(params); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	// try to decrypt
	ks := nodectx.GetNodeCtx().Keystore
	if err := ks.Restore(params.Seeds, params.Keystore, params.Config, params.Path, params.Password); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("restore failed: %s", err)})
	}

	result := RestoreResult{
		Path: params.Path,
	}
	return c.JSON(http.StatusOK, result)
}

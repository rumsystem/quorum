package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
)

type SchemaListItem struct {
	Type      string `validate:"required"`
	Rule      string `validate:"required"`
	TimeStamp int64  `validate:"required"`
}

// @Tags Group
// @Summary GetGroupAppSchema
// @Description Get group schema
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {object} []SchemaListItem
// @Router /api/v1/group/{group_id}/app/schema [get]

func (h *Handler) GetGroupAppSchema(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[groupid]
	if !ok {
		return rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	}

	schemaList, err := group.GetSchemas()
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	schemaResultList := []*SchemaListItem{}
	for _, schema := range schemaList {
		var item *SchemaListItem
		item = &SchemaListItem{}
		item.Type = schema.Type
		item.Rule = schema.Rule
		item.TimeStamp = schema.TimeStamp
		schemaResultList = append(schemaResultList, item)
	}

	return c.JSON(http.StatusOK, schemaResultList)
}

package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/chain"
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

	output := make(map[string]string)
	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[groupid]
	if !ok {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}

	schemaList, err := group.GetSchemas()
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
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

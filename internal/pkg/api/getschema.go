package api

import (
	"github.com/labstack/echo/v4"
)

type SchemaListItem struct {
	schema string
}

func (h *Handler) GetGroupAppSchema(c echo.Context) (err error) {
	/*
		output := make(map[string]string)
		groupid := c.Param("group_id")
		if groupid == "" {
			output[ERROR_INFO] = "group_id can't be nil."
			return c.JSON(http.StatusBadRequest, output)
		}

		if group, ok := chain.GetChainCtx().Groups[groupid]; ok {
			smaList, err := chain.GetDbMgr().GetSchemaByGroup(group.Item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}

			var smaResultList []*SchemaListItem
			for _, sma := range smaList {
				var item *SchemaListItem
				item = &SchemaListItem{}
				item.schema = sma.SchemaJson
				smaResultList = append(smaResultList, item)
			}

			return c.JSON(http.StatusOK, smaResultList)
		} else {
			output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
			return c.JSON(http.StatusBadRequest, output)
		}
	*/

	return nil
}

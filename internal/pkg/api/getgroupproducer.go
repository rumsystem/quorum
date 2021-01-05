package api

import (
	"fmt"
	"net/http"

	"github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type ProducerListItem struct {
	ProducerPubkey string
}

// @Tags Management
// @Summary GetGroupProducers
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} ProducerListItem
// @Router /api/v1/group/{group_id}/producers [get]
func (h *Handler) GetGroupProducers(c echo.Context) (err error) {

	output := make(map[string]string)
	groupid := c.Param("group_id")

	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetNodeCtx().Groups[groupid]; ok {
		prdList, err := group.GetProducers()
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var prdResultList []*ProducerListItem
		for _, prd := range prdList {
			var item *ProducerListItem
			item = &ProducerListItem{}
			item.ProducerPubkey = prd.ProducerPubkey
			prdResultList = append(prdResultList, item)
		}

		return c.JSON(http.StatusOK, prdResultList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

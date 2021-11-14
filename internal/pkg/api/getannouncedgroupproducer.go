package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/chain"
)

type AnnouncedProducerListItem struct {
	AnnouncedPubkey string `validate:"required"`
	AnnouncerSign   string `validate:"required"`
	Result          string `validate:"required"`
	Action          string `validate:"required"`
	Memo            string `validate:"required"`
	TimeStamp       int64  `validate:"required"`
}

// @Tags User
// @Summary GetAnnouncedGroupProducer
// @Description Get the list of group producers
// @Produce json
// @Param group_id path string true "Group Id"
// @Success 200 {array} AnnouncedUserListItem
// @Router /api/v1/group/{group_id}/announced/producers [get]
func (h *Handler) GetAnnouncedGroupProducer(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")

	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := group.GetAnnouncedProducers()
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		prdResultList := []*AnnouncedProducerListItem{}
		for _, prd := range prdList {
			var item *AnnouncedProducerListItem
			item = &AnnouncedProducerListItem{}
			item.AnnouncedPubkey = prd.SignPubkey
			item.AnnouncerSign = prd.AnnouncerSignature
			item.Result = prd.Result.String()
			item.Action = prd.Action.String()
			item.TimeStamp = prd.TimeStamp
			item.Memo = prd.Memo
			prdResultList = append(prdResultList, item)
		}

		return c.JSON(http.StatusOK, prdResultList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}

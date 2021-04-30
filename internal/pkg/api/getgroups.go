package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
)

type groupInfo struct {
	OwnerPubKey    string `json:"OwnerPubKey"`
	GroupId        string `json:"GroupId"`
	GroupName      string `json:"GroupName"`
	LastUpdate     int64  `json:"LastUpdate"`
	LatestBlockNum int64  `json:"LatestBlockNum"`
	LatestBlockId  string `json:"LatestBlockId"`
	GroupStatus    string `json:"GroupStatus"`
}

type groupInfoList struct {
	GroupInfos []*groupInfo `json:"groups"`
}

func (h *Handler) GetGroups(c echo.Context) (err error) {
	var groups []*groupInfo
	for _, value := range chain.GetChainCtx().Groups {
		var group *groupInfo
		group = &groupInfo{}

		group.OwnerPubKey = value.Item.OwnerPubKey
		group.GroupId = value.Item.GroupId
		group.GroupName = value.Item.GroupName
		group.LastUpdate = value.Item.LastUpdate
		group.LatestBlockNum = value.Item.LatestBlockNum
		group.LatestBlockId = value.Item.LatestBlockId
		if value.Status == chain.GROUP_CLEAN {
			group.GroupStatus = "GROUP_READY"
		} else {
			group.GroupStatus = "GROUP_SYNCING"
		}
		groups = append(groups, group)
	}
	return c.JSON(http.StatusOK, &groupInfoList{groups})
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/peer"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
)

type groupNetworkInfo struct {
	GroupId   string    `json:"GroupId"`
	GroupName string    `json:"GroupName"`
	Peers     []peer.ID `json:"Peers"`
}

func (h *Handler) GetNetwork(c echo.Context) (err error) {
	result := make(map[string]interface{})
	groupnetworklist := []*groupNetworkInfo{}
	for _, group := range chain.GetChainCtx().Groups {
		groupnetwork := &groupNetworkInfo{}
		groupnetwork.GroupId = group.Item.GroupId
		groupnetwork.GroupName = group.Item.GroupName
		groupnetwork.Peers = chain.GetChainCtx().ListGroupPeers(group.Item.GroupId)
		groupnetworklist = append(groupnetworklist, groupnetwork)
	}
	result["groups"] = groupnetworklist
	return c.JSON(http.StatusOK, result)
}

package api

import (
	"net/http"

	"github.com/huo-ju/quorum/internal/pkg/options"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
)

type groupNetworkInfo struct {
	GroupId   string    `json:"GroupId"`
	GroupName string    `json:"GroupName"`
	Peers     []peer.ID `json:"Peers"`
}

func (h *Handler) GetNetwork(nodehost *host.Host, nodeinfo *p2p.NodeInfo, nodeopt *options.NodeOptions, ethaddr string) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := make(map[string]interface{})
		node := make(map[string]interface{})
		groupnetworklist := []*groupNetworkInfo{}
		for _, group := range chain.GetChainCtx().Groups {
			groupnetwork := &groupNetworkInfo{}
			groupnetwork.GroupId = group.Item.GroupId
			groupnetwork.GroupName = group.Item.GroupName
			groupnetwork.Peers = chain.GetChainCtx().ListGroupPeers(group.Item.GroupId)
			groupnetworklist = append(groupnetworklist, groupnetwork)
		}
		node["peerid"] = (*nodehost).ID()
		node["ethaddr"] = ethaddr
		node["nat_type"] = nodeinfo.NATType.String()
		node["nat_enabled"] = nodeopt.EnableNat
		node["addrs"] = (*nodehost).Addrs()
		result["groups"] = groupnetworklist
		result["node"] = node
		return c.JSON(http.StatusOK, result)
	}
}

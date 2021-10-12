package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

type groupNetworkInfo struct {
	GroupId   string    `json:"GroupId"`
	GroupName string    `json:"GroupName"`
	Peers     []peer.ID `json:"Peers"`
}

type NetworkInfo struct {
	Peerid     string                 `json:"peerid"`
	Ethaddr    string                 `json:"ethaddr"`
	NatType    string                 `json:"nat_type"`
	NatEnabled bool                   `json:"nat_enabled"`
	Addrs      []maddr.Multiaddr      `json:"addrs"`
	Groups     []*groupNetworkInfo    `json:"groups"`
	Node       map[string]interface{} `json:"node"`
}

// @Tags Node
// @Summary GetNetwork
// @Description Get node's network information
// @Produce json
// @Success 200 {object} NetworkInfo
// @Router /api/v1/network [get]
func (h *Handler) GetNetwork(nodehost *host.Host, nodeinfo *p2p.NodeInfo, nodeopt *options.NodeOptions, ethaddr string) echo.HandlerFunc {

	return func(c echo.Context) error {
		result := &NetworkInfo{}
		node := make(map[string]interface{})
		groupnetworklist := []*groupNetworkInfo{}
		groupmgr := chain.GetGroupMgr()
		for _, group := range groupmgr.Groups {
			groupnetwork := &groupNetworkInfo{}
			groupnetwork.GroupId = group.Item.GroupId
			groupnetwork.GroupName = group.Item.GroupName
			groupnetwork.Peers = nodectx.GetNodeCtx().ListGroupPeers(group.Item.GroupId)
			groupnetworklist = append(groupnetworklist, groupnetwork)
		}
		result.Peerid = (*nodehost).ID().Pretty()
		result.Ethaddr = ethaddr
		result.NatType = nodeinfo.NATType.String()
		result.NatEnabled = nodeopt.EnableNat
		result.Addrs = (*nodehost).Addrs()

		result.Groups = groupnetworklist
		result.Node = node
		return c.JSON(http.StatusOK, result)
	}
}

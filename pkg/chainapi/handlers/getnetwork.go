package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type groupNetworkInfo struct {
	GroupId   string    `json:"GroupId" validate:"required,uuid4" example:"997ce496-661b-457b-8c6a-f57f6d9862d0"`
	GroupName string    `json:"GroupName" validate:"required" example:"pb_group_1"`
	Peers     []peer.ID `json:"Peers" validate:"required" example:"16Uiu2HAkuXLC2hZTRbWToCNztyWB39KDi8g66ou3YrSzeTbsWsFG,16Uiu2HAm8XVpfQrJYaeL7XtrHC3FvfKt2QW7P8R3MBenYyHxu8Kk"`
}

type NetworkInfo struct {
	Peerid     string                 `json:"peer_id" validate:"required" example:"16Uiu2HAm8XVpfQrJYaeL7XtrHC3FvfKt2QW7P8R3MBenYyHxu8Kk"`
	Ethaddr    string                 `json:"eth_addr" validate:"required" example:"0x4daD72e78c3537a8852ca7b3d1742Dd42c30441A"`
	NatType    string                 `json:"nat_type" validate:"required" example:"Public"`
	NatEnabled bool                   `json:"nat_enabled" validate:"required" example:"true"`
	Addrs      []maddr.Multiaddr      `json:"addrs" validate:"required"` // Example: ["/ip4/192.168.20.17/tcp/7002", "/ip4/127.0.0.1/tcp/7002"]
	Groups     []*groupNetworkInfo    `json:"groups" validate:"required"`
	Node       map[string]interface{} `json:"node" validate:"required"`
}

func (n *NetworkInfo) UnmarshalJSON(data []byte) error {
	type Alias NetworkInfo
	network := &struct {
		Addrs []string `json:"addrs"`
		*Alias
	}{
		Alias: (*Alias)(n),
	}

	if err := json.Unmarshal(data, &network); err != nil {
		return err
	}

	addrs, err := utils.StringsToAddrs(network.Addrs)
	if err != nil {
		return err
	}
	n.Addrs = addrs

	return nil
}

func GetNetwork(nodehost *host.Host, nodeinfo *p2p.NodeInfo, nodeopt *options.NodeOptions, ethaddr string) (*NetworkInfo, error) {
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

	_, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("json.Marshal failed: %s", err)
	}

	return result, nil
}

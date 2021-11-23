package handlers

import (
	"fmt"
	"strings"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type NodeInfo struct {
	NodeID        string              `json:"node_id" validate:"required"`
	NodePublickey string              `json:"node_publickey" validate:"required"`
	NodeStatus    string              `json:"node_status" validate:"required"`
	NodeType      string              `json:"node_type" validate:"required"`
	NodeVersion   string              `json:"node_version" validate:"required"`
	Peers         map[string][]string `json:"peers" validate:"required"`
}

func updateNodeStatus(nodenetworkname string) {
	peersprotocol := nodectx.GetNodeCtx().PeersProtocol()
	networknamewithprefix := fmt.Sprintf("%s/%s", p2p.ProtocolPrefix, nodenetworkname)
	for protocol, peerlist := range *peersprotocol {
		if strings.HasPrefix(protocol, networknamewithprefix) {
			if len(peerlist) > 0 {
				nodectx.GetNodeCtx().UpdateOnlineStatus(nodectx.NODE_ONLINE)
				return
			}
		}
	}
	if nodectx.GetNodeCtx().Status != nodectx.NODE_OFFLINE {
		nodectx.GetNodeCtx().UpdateOnlineStatus(nodectx.NODE_OFFLINE)
	}
}

func GetNodeInfo(networkName string) (*NodeInfo, error) {

	var info NodeInfo

	info.NodeVersion = nodectx.GetNodeCtx().Version + " - " + utils.GitCommit
	info.NodeType = "peer"
	updateNodeStatus(networkName)

	if nodectx.GetNodeCtx().Status == nodectx.NODE_ONLINE {
		info.NodeStatus = "NODE_ONLINE"
	} else {
		info.NodeStatus = "NODE_OFFLINE"
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(nodectx.GetNodeCtx().PublicKey)
	if err != nil {
		return nil, err
	}

	info.NodePublickey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	info.NodeID = nodectx.GetNodeCtx().PeerId.Pretty()

	peers := nodectx.GetNodeCtx().PeersProtocol()
	info.Peers = *peers

	return &info, nil
}

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
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

// @Tags Node
// @Summary GetNodeInfo
// @Description Return the node info
// @Produce json
// @Success 200 {object} NodeInfo
// @Router /api/v1/node [get]
func (h *Handler) GetNodeInfo(c echo.Context) (err error) {
	output := make(map[string]interface{})
	var info NodeInfo

	info.NodeVersion = nodectx.GetNodeCtx().Version + " - " + h.GitCommit
	info.NodeType = "peer"
	updateNodeStatus(h.Node.NetworkName)
	if nodectx.GetNodeCtx().Status == nodectx.NODE_ONLINE {
		info.NodeStatus = "NODE_ONLINE"
	} else {
		info.NodeStatus = "NODE_OFFLINE"
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(nodectx.GetNodeCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	info.NodePublickey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	info.NodeID = nodectx.GetNodeCtx().PeerId.Pretty()

	peers := nodectx.GetNodeCtx().PeersProtocol()
	info.Peers = *peers

	return c.JSON(http.StatusOK, info)
}

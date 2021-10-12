package api

import (
	"fmt"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"net/http"
	"strings"
)

type NodeInfo struct {
	Node_publickey string `json:"node_publickey"`
	Node_status    string `json:"node_status"`
	Node_version   string `json:"node_version"`
	User_id        string `json:"user_id"`
}

func updateNodeStatus() {
	peersprotocol := nodectx.GetNodeCtx().PeersProtocol()
	for protocol, peerlist := range *peersprotocol {

		if strings.HasPrefix(protocol, fmt.Sprintf("%s/meshsub", p2p.ProtocolPrefix)) {
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

	output[NODE_VERSION] = nodectx.GetNodeCtx().Version + " - " + h.GitCommit
	output[NODETYPE] = "peer"
	updateNodeStatus()
	if nodectx.GetNodeCtx().Status == nodectx.NODE_ONLINE {
		output[NODE_STATUS] = "NODE_ONLINE"
	} else {
		output[NODE_STATUS] = "NODE_OFFLINE"
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(nodectx.GetNodeCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	output[NODE_PUBKEY] = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	output[NODE_ID] = nodectx.GetNodeCtx().PeerId.Pretty()

	peers := nodectx.GetNodeCtx().PeersProtocol()
	output[PEERS] = *peers

	return c.JSON(http.StatusOK, output)
}

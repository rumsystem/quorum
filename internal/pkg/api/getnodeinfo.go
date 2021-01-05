package api

import (
	"fmt"
	"github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
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
	peersprotocol := chain.GetNodeCtx().PeersProtocol()
	for protocol, peerlist := range *peersprotocol {

		if strings.HasPrefix(protocol, fmt.Sprintf("%s/meshsub", p2p.ProtocolPrefix)) {
			if len(peerlist) > 0 {
				chain.GetNodeCtx().UpdateOnlineStatus(chain.NODE_ONLINE)
				return
			}
		}
	}
	if chain.GetNodeCtx().Status != chain.NODE_OFFLINE {
		chain.GetNodeCtx().UpdateOnlineStatus(chain.NODE_OFFLINE)
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

	output[NODE_VERSION] = chain.GetNodeCtx().Version + " - " + h.GitCommit
	output[NODETYPE] = "peer"
	updateNodeStatus()
	if chain.GetNodeCtx().Status == chain.NODE_ONLINE {
		output[NODE_STATUS] = "NODE_ONLINE"
	} else {
		output[NODE_STATUS] = "NODE_OFFLINE"
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetNodeCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	output[NODE_PUBKEY] = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	output[NODE_ID] = chain.GetNodeCtx().PeerId.Pretty()

	peers := chain.GetNodeCtx().PeersProtocol()
	output[PEERS] = *peers

	return c.JSON(http.StatusOK, output)
}

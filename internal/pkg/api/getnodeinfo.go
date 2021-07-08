package api

import (
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type NodeInfo struct {
	Node_publickey string `json:"node_publickey"`
	Node_status string `json:"node_status"`
	Node_version string `json:"node_version"`
	User_id string `json:"user_id"`
}

// @Tags Node
// @Summary GetNodeInfo
// @Description Return the node info
// @Produce json
// @Success 200 {object} NodeInfo
// @Router /v1/node [get]
func (h *Handler) GetNodeInfo(c echo.Context) (err error) {
	output := make(map[string]interface{})

	output[NODE_VERSION] = chain.GetChainCtx().Version
	output[NODETYPE] = "peer"

	if chain.GetChainCtx().Status == 0 {
		output[NODE_STATUS] = "NODE_ONLINE"
	} else {
		output[NODE_STATUS] = "NODE_OFFLINE"
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	output[NODE_PUBKEY] = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	output[USER_ID] = chain.GetChainCtx().PeerId.Pretty()

	peers := chain.GetChainCtx().Peers()
	output[PEERS] = *peers

	return c.JSON(http.StatusOK, output)
}

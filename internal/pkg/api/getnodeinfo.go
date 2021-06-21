package api

import (
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

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
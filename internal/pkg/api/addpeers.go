package api

import (
	"fmt"
	"net/http"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
)

type AddPeerResult struct {
	SuccCount int `json:"succ_count"`
	ErrCount  int `json:"err_count"`
}

// @Tags Node
// @Summary AddPeer
// @Description Connect to peers
// @Accept json
// @Produce json
// @Param data body []string true "Peers List"
// @Success 200 {object} AddPeerResult
// @Router /api/v1/network/peers [post]
func (h *Handler) AddPeers(c echo.Context) (err error) {
	output := make(map[string]string)
	input := []string{}
	peerserr := make(map[string]string)

	if err = c.Bind(&input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	peersaddrinfo := []peer.AddrInfo{}
	for _, addr := range input {
		ma, err := maddr.NewMultiaddr(addr)
		if err != nil {
			peerserr[addr] = fmt.Sprintf("%s", err)
			continue
		}
		addrinfo, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			peerserr[addr] = fmt.Sprintf("%s", err)
			continue
		}
		peersaddrinfo = append(peersaddrinfo, *addrinfo)
	}

	result := &AddPeerResult{SuccCount: 0, ErrCount: len(peerserr)}

	if len(peersaddrinfo) > 0 {
		count := chain.GetNodeCtx().AddPeers(peersaddrinfo)
		result.SuccCount = count
	}
	return c.JSON(http.StatusOK, result)
}

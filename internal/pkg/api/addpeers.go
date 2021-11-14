package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type AddPeerParam []string

type AddPeerResult struct {
	SuccCount int `json:"succ_count"`
	ErrCount  int `json:"err_count"`
}

// @Tags Node
// @Summary AddPeers
// @Description Connect to peers
// @Accept json
// @Produce json
// @Param data body []string true "Peers List"
// @Success 200 {object} AddPeerResult
// @Router /api/v1/network/peers [post]
func (h *Handler) AddPeers(c echo.Context) (err error) {
	var input AddPeerParam
	output := make(map[string]string)
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
		count := nodectx.GetNodeCtx().AddPeers(peersaddrinfo)
		result.SuccCount = count
	}
	return c.JSON(http.StatusOK, result)
}

package api

import (
	"fmt"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"net/http"
)

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
		if len(peersaddrinfo) > 0 {
			count := chain.GetChainCtx().AddPeers(peersaddrinfo)
			output["count"] = fmt.Sprintf("%d", count)
		}
	}
	return c.JSON(http.StatusOK, output)
}

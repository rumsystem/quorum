package api

import (
	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"net/http"
)

type PubkeyParam struct {
	EncodedPubkey string `from:"encoded_pubkey" json:"encoded_pubkey" validate:"required"`
}

// @Tags Tools
// @Summary PubkeyToEthaddr
// @Description Convert a based64 encoded libp2p pubkey to the eth address
// @Accept json
// @Produce json
// @Param data body PubkeyParam true "PubkeyParam"
// @Success 200 {object} map[string]string
// @Router /api/v1/tools/pubkeytoaddr [post]
func (h *Handler) PubkeyToEthaddr(c echo.Context) (err error) {
	var input PubkeyParam
	output := make(map[string]string)

	if err = c.Bind(&input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(input.EncodedPubkey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	output["addr"] = ethaddr
	return c.JSON(http.StatusOK, output)
}

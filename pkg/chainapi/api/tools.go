package api

import (
	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"net/http"
)

type PubkeyParam struct {
	EncodedPubkey string `from:"encoded_pubkey" json:"encoded_pubkey" validate:"required"`
}

type PubkeyToEthaddrResult struct {
	Addr string `json:"addr"`
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
	cc := c.(*utils.CustomContext)

	input := new(PubkeyParam)
	if err := cc.BindAndValidate(input); err != nil {
		return err
	}

	ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(input.EncodedPubkey)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	result := PubkeyToEthaddrResult{Addr: ethaddr}
	return c.JSON(http.StatusOK, result)
}

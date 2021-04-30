package api

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/go-playground/validator/v10"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type RmGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

func (h *Handler) RmGroup(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(RmGroupParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	shouldRemove := false
	if group, ok := chain.GetChainCtx().Groups[params.GroupId]; ok {
		err := group.DelGrp()

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		shouldRemove = true
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}

	if shouldRemove {
		delete(chain.GetChainCtx().Groups, params.GroupId)
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var buffer bytes.Buffer
	buffer.Write(pubkeybytes)
	buffer.Write([]byte(params.GroupId))
	hash := chain.Hash(buffer.Bytes())
	signature, err := chain.Sign(hash)

	output[GROUP_ID] = params.GroupId
	output[GROUP_OWNER_PUBKEY] = p2pcrypto.ConfigEncodeKey(pubkeybytes)

	output[SIGNATURE] = fmt.Sprintf("%x", signature)

	return c.JSON(http.StatusOK, output)
}

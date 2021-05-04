package api

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type RmBlkGrpUserParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
	UserId  string `from:"user_id" json:"user_id" validate:"required"`
	Memo    string `from:"memo" json:"memo" validate:"required"`
}

type RmBlkGrpUserResult struct {
	GroupId    string `json:"group_id"`
	UserId     string `json:"user_id"`
	NodePubkey string `json:"node_pubkey"`
	Sign       string `json:"sign"`
}

func (h *Handler) RmBlkedGrpUser(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(BlockGrpUserParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	err = chain.GetDbMgr().RmBlkLlist(params.GroupId, params.UserId)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)

	var buffer bytes.Buffer
	buffer.Write([]byte(params.GroupId))
	buffer.Write([]byte(params.UserId))
	buffer.Write(pubkeybytes)
	hash := chain.Hash(buffer.Bytes())
	signature, err := chain.Sign(hash)

	rmblkGrpUserResult := &RmBlkGrpUserResult{GroupId: params.GroupId, UserId: params.UserId, NodePubkey: p2pcrypto.ConfigEncodeKey(pubkeybytes), Sign: fmt.Sprintf("%x", signature)}

	return c.JSON(http.StatusOK, rmblkGrpUserResult)
}

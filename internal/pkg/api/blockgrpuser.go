package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type BlockGrpUserParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
	UserId  string `from:"user_id" json:"user_id" validate:"required"`
	Memo    string `from:"memo" json:"memo" validate:"required"`
}

type BlockGrpUserResult struct {
	GroupId     string `json:"group_id"`
	UserId      string `json:"user_id"`
	OwnerPubkey string `json:"owner_pubkey"`
	Sign        string `json:"sign"`
}

func (h *Handler) BlkGrpUser(c echo.Context) (err error) {
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

	var item *quorumpb.BlockListItem
	item = &quorumpb.BlockListItem{}

	item.GroupId = params.GroupId
	item.UserId = params.UserId

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
	item.OwnerPubkey = p2pcrypto.ConfigEncodeKey(pubkeybytes)

	var buffer bytes.Buffer
	buffer.Write([]byte(params.GroupId))
	buffer.Write([]byte(params.UserId))
	buffer.Write(pubkeybytes)
	hash := chain.Hash(buffer.Bytes())
	signature, err := chain.Sign(hash)

	item.OwnerSign = fmt.Sprintf("%x", signature)
	item.Memo = params.Memo
	item.TimeStamp = time.Now().UnixNano()

	err = chain.GetDbMgr().AddBlkList(item)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	blockGrpUserResult := &BlockGrpUserResult{GroupId: params.GroupId, UserId: params.UserId, OwnerPubkey: p2pcrypto.ConfigEncodeKey(pubkeybytes), Sign: fmt.Sprintf("%x", signature)}
	return c.JSON(http.StatusOK, blockGrpUserResult)
}

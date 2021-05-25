package api

import (
	//"encoding/json"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type JoinGroupParam struct {
	GroupId      string          `from:"group_id" json:"group_id" validate:"required"`
	GroupName    string          `from:"group_name" json:"group_name" validate:"required"`
	OwnerPubKey  string          `from:"owner_pubkey" json:"owner_pubkey" validate:"required"`
	GenesisBlock *quorumpb.Block `from:"genesis_block" json:"genesis_block" validate:"required"`
}

func (h *Handler) JoinGroup(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(JoinGroupParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err != nil {
		output[ERROR_INFO] = "unmarshal genesis block failed with msg:" + err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var item *quorumpb.GroupItem
	item = &quorumpb.GroupItem{}

	item.OwnerPubKey = params.OwnerPubKey
	item.GroupId = params.GroupId
	item.GroupName = params.GroupName
	item.LatestBlockId = params.GenesisBlock.Cid
	item.LatestBlockNum = params.GenesisBlock.BlockNum
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = params.GenesisBlock

	var group *chain.Group
	group = &chain.Group{}

	err = group.CreateGrp(item)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	err = chain.GetChainCtx().JoinGroupChannel(item.GroupId, context.Background())
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	err = group.StartSync()
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	chain.GetChainCtx().Groups[group.Item.GroupId] = group

	genesisBlockBytes, err := json.Marshal(item.GenesisBlock)
	if err != nil {
		output[ERROR_INFO] = "marshal genesis block failed with msg:" + err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var buffer bytes.Buffer
	buffer.Write(genesisBlockBytes)
	buffer.Write(pubkeybytes)
	buffer.Write([]byte(item.GroupId))
	hash := chain.Hash(buffer.Bytes())
	signature, err := chain.Sign(hash)

	output[GROUP_ID] = params.GroupId
	output[SIGNATURE] = fmt.Sprintf("%x", signature)

	return c.JSON(http.StatusOK, output)
}

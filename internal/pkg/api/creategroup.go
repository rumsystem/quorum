package api

import (
	"encoding/json"
	"net/http"
	"time"
	//"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	guuid "github.com/google/uuid"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type CreateGroupParam struct {
	GroupName string `from:"group_name" json:"group_name" validate:"required,max=20,min=5"`
}

func (h *Handler) CreateGroup(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(CreateGroupParam)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupid := guuid.New()
	genesisBlock := chain.CreateGenesisBlock(groupid.String())

	b, err := json.Marshal(genesisBlock)
	if err != nil {
		output[ERROR_INFO] = "create genesis block failed with msg:" + err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	bs := string(b)

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	output[GROUP_ID] = groupid.String()
	output[GROUP_NAME] = params.GroupName
	output[GROUP_OWNER_PUBKEY] = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	output[GENESIS_BLOCK] = bs
	output[SIGNATURE] = "owner_signature" //Sign for all above items

	//create local group
	var item *chain.GroupItem
	item = &chain.GroupItem{}

	item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	item.GroupId = groupid.String()
	item.GroupName = params.GroupName
	item.LatestBlockId = genesisBlock.Cid
	item.LatestBlockNum = genesisBlock.BlockNum
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = genesisBlock

	var group *chain.Group
	group = &chain.Group{}

	err = group.CreateGrp(item)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	chain.GetChainCtx().Groups[group.Item.GroupId] = group

	return c.JSON(http.StatusOK, output)
}

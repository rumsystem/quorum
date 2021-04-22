package api

import (
	//"encoding/json"
	"net/http"
	"time"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type JoinGroupParam struct {
	GroupId      string       `from:"group_id" json:"group_id" validate:"required`
	GroupName    string       `from:"group_name" json:"group_name" validate:"required"`
	OwnerPubKey  string       `from:"owner_pubkey" json:"owner_pubkey" validate:"required"`
	GenesisBlock *chain.Block `from:"genesis_block" json:"genesis_block" validate:"required"`
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

	//var genesisBlock chain.Block

	//b := []byte(params.GenesisBlock)
	//err = json.Unmarshal(b, &genesisBlock)
	if err != nil {
		output[ERROR_INFO] = "unmarshal genesis block failed with msg:" + err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var item *chain.GroupItem
	item = &chain.GroupItem{}

	item.OwnerPubKey = params.OwnerPubKey
	item.GroupId = params.GroupId
	item.GroupName = params.GroupName
	item.LatestBlockId = params.GenesisBlock.Cid
	item.LatestBlockNum = params.GenesisBlock.BlockNum
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = *params.GenesisBlock

	var group *chain.Group
	group = &chain.Group{}

	err = group.CreateGrp(item)

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

	output[GROUP_ID] = params.GroupId
	output[SIGNATURE] = "Owner Signature"
	return c.JSON(http.StatusOK, output)
}

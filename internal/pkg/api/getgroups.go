package api

import (
	"encoding/json"
	//"fmt"
	"github.com/labstack/echo/v4"
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type groupInfo struct {
	OwnerPubKey    string
	GroupId        string
	GroupName      string
	LastUpdate     int64
	LatestBlockNum int64
	LatestBlockId  string
	GroupStatus    string
}

func (h *Handler) GetGroups(c echo.Context) (err error) {
	output := make(map[string]string)
	var groups []*groupInfo
	for _, value := range chain.GetChainCtx().Groups {
		var group *groupInfo
		group = &groupInfo{}

		pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		group.OwnerPubKey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
		group.GroupId = value.Item.GroupId
		group.LastUpdate = value.Item.LastUpdate
		group.LatestBlockNum = value.Item.LatestBlockNum
		group.LatestBlockId = value.Item.LatestBlockId
		if value.Status == chain.GROUP_CLEAN {
			group.GroupStatus = "GROUP_READY"
		} else {
			group.GroupStatus = "GROUP_SYNCING"
		}
		groups = append(groups, group)
	}

	result, err := json.Marshal(groups)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	output[GROUP_ITEMS] = string(result)
	return c.JSON(http.StatusOK, output)
}

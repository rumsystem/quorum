package api

import (
	"github.com/labstack/echo/v4"
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type groupInfo struct {
	OwnerPubKey    string `json:"OwnerPubKey"`
	GroupId        string `json:"GroupId"`
	GroupName      string `json:"GroupName"`
	LastUpdate     int64  `json:"LastUpdate"`
	LatestBlockNum int64  `json:"LatestBlockNum"`
	LatestBlockId  string `json:"LatestBlockId"`
	GroupStatus    string `json:"GroupStatus"`
}

type groupInfoList struct {
	GroupInfos []*groupInfo `json:"groups"`
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
	return c.JSON(http.StatusOK, &groupInfoList{groups})
}

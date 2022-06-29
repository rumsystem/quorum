package nodesdkapi

import (
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GroupInfo struct {
	GroupId        string   `json:"group_id" validate:"required,uuid4"`
	GroupName      string   `json:"group_name" validate:"required"`
	SignAlias      string   `json:"sign_alias" validate:"required"`
	EncryptAlias   string   `json:"encrypt_alias" validate:"required"`
	UserEthaddr    string   `json:"user_eth_addr" validate:"required"`
	ConsensusType  string   `json:"consensus_type" validate:"required"`
	EncryptionType string   `json:"encryption_type" validate:"required"`
	CipherKey      string   `json:"cipher_key" validate:"required"`
	AppKey         string   `json:"app_key" validate:"required"`
	LastUpdated    int64    `json:"last_updated" validate:"required"`
	HighestHeight  int64    `json:"highest_height" validate:"required"`
	HighestBlockId string   `json:"highest_block_id" validate:"required,uuid4"`
	ChainApis      []string `json:"chain_apis" validate:"required,uuid4"`
}

type GroupInfoList struct {
	GroupInfos []*GroupInfo `json:"groups"`
}

// for sort
func (s *GroupInfoList) Len() int {
	return len(s.GroupInfos)
}

func (s *GroupInfoList) Swap(i, j int) {
	s.GroupInfos[i], s.GroupInfos[j] = s.GroupInfos[j], s.GroupInfos[i]
}
func (s *GroupInfoList) Less(i, j int) bool {
	return s.GroupInfos[i].GroupName < s.GroupInfos[j].GroupName
}

// end

func (h *NodeSDKHandler) GetAllGroups() echo.HandlerFunc {
	return func(c echo.Context) error {
		var groups []*GroupInfo
		nodesdkGroupItems, err := nodesdkctx.GetCtx().GetChainStorage().GetAllGroupsV2()
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		for _, groupItem := range nodesdkGroupItems {
			var groupInfo *GroupInfo
			groupInfo = &GroupInfo{}
			groupInfo.GroupId = groupItem.Group.GroupId
			groupInfo.GroupName = groupItem.Group.GroupName
			groupInfo.SignAlias = groupItem.SignAlias
			groupInfo.EncryptAlias = groupItem.EncryptAlias

			ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(groupItem.Group.UserSignPubkey)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}
			groupInfo.UserEthaddr = ethaddr
			groupInfo.ConsensusType = groupItem.Group.ConsenseType.String()
			groupInfo.EncryptionType = groupItem.Group.EncryptType.String()
			groupInfo.CipherKey = groupItem.Group.CipherKey
			groupInfo.AppKey = groupItem.Group.AppKey
			groupInfo.LastUpdated = groupItem.Group.LastUpdate
			groupInfo.HighestHeight = groupItem.Group.HighestHeight
			groupInfo.HighestBlockId = groupItem.Group.HighestBlockId
			groupInfo.ChainApis = groupItem.ApiUrl
			groups = append(groups, groupInfo)
		}
		ret := GroupInfoList{groups}
		sort.Sort(&ret)
		return c.JSON(http.StatusOK, ret)
	}
}

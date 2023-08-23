package nodesdkapi

import (
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GroupInfo struct {
	GroupId       string   `json:"group_id" validate:"required,uuid4"`
	GroupName     string   `json:"group_name" validate:"required"`
	SignAlias     string   `json:"sign_alias" validate:"required"`
	UserEthaddr   string   `json:"user_eth_addr" validate:"required"`
	ConsensusType string   `json:"consensus_type" validate:"required"`
	SyncType      string   `json:"sync_type" validate:"required"`
	CipherKey     string   `json:"cipher_key" validate:"required"`
	AppId         string   `json:"app_id" validate:"required"`
	AppName       string   `json:"app_name" validate:"required"`
	LastUpdated   int64    `json:"last_updated" validate:"required"`
	Epoch         int64    `json:"epoch" validate:"required"`
	ChainApis     []string `json:"chain_apis" validate:"required,uuid4"`
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
			ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(groupItem.Group.UserSignPubkey)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}
			groupInfo.UserEthaddr = ethaddr
			groupInfo.ConsensusType = groupItem.Group.ConsenseType.String()
			groupInfo.SyncType = groupItem.Group.SyncType.String()
			groupInfo.CipherKey = groupItem.Group.CipherKey
			groupInfo.AppId = groupItem.Group.AppId
			groupInfo.AppName = groupItem.Group.AppName
			groupInfo.LastUpdated = groupItem.Group.LastUpdate
			groupInfo.ChainApis = groupItem.ApiUrl
			groups = append(groups, groupInfo)
		}
		ret := GroupInfoList{groups}
		sort.Sort(&ret)
		return c.JSON(http.StatusOK, ret)
	}
}

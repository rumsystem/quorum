package api

import (
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/peer"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type GroupInfo struct {
	GroupId       string `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	GroupName     string `json:"group_name" validate:"required" example:"demo-app"`
	OwnerPubKey   string `json:"owner_pubkey" validate:"required" example:"CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg=="`
	ConsensusType string `json:"consensus_type" validate:"required" example:"POA"`
	AuthType      string `json:"auth_type" validate:"required" example:"PUBLIC"`
	CipherKey     string `json:"cipher_key" validate:"required" example:"58044622d48c4d91932583a05db3ff87f29acacb62e701916f7f0bbc6e446e5d"`
	AppId         string `json:"app_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	AppName       string `json:"app_name" validate:"required" example:"demo-app"`
	CurrtTopBlock uint64 `json:"currt_top_block" validate:"required" example:"0"`
	LastUpdated   int64  `json:"last_updated" validate:"required" example:"1633022375303983600"`

	Peers []peer.ID `json:"peers" validate:"required" example:"16Uiu2HAkuXLC2hZTRbWToCNztyWB39KDi8g66ou3YrSzeTbsWsFG,16Uiu2HAm8XVpfQrJYaeL7XtrHC3FvfKt2QW7P8R3MBenYyHxu8Kk"`
}

//move to GroupServiceInfo
//UserPubkey    string `json:"user_pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
//UserEthaddr   string `json:"user_eth_addr" validate:"required" example:"0495180230ae0f585ca0b4fc0767e616eaed45e400f470ed50c91668e1ed76c278b7fc5a129ff154c6b200a26cc78b7b4acc5b3915cdf66286c942aa5b65166ff5"`
//RexSyncerStatus string             `json:"rex_syncer_status" validate:"required" example:"IDLE"`
//RexSyncerResult *def.RexSyncResult `json:"rex_Syncer_result" validate:"required"`

type GroupInfoList struct {
	GroupInfos []*GroupInfo `json:"groups"`
}

func (s *GroupInfoList) Len() int { return len(s.GroupInfos) }
func (s *GroupInfoList) Swap(i, j int) {
	s.GroupInfos[i], s.GroupInfos[j] = s.GroupInfos[j], s.GroupInfos[i]
}

func (s *GroupInfoList) Less(i, j int) bool {
	return s.GroupInfos[i].GroupName < s.GroupInfos[j].GroupName
}

// @Tags Groups
// @Summary GetGroups
// @Description Get group info of all joined groups
// @Produce json
// @Success 200 {object} GroupInfoList
// @Router /api/v1/groups [get]
func (h *Handler) ListLocalGroups(c echo.Context) (err error) {
	groupIfaces, err := chain.GetGroupMgr().GetLocalGroupIfaces()
	if err != nil {
		return err
	}
	var groupInfos []*GroupInfo
	for _, groupIface := range groupIfaces {
		groupInfo, err := getGroupInfo(groupIface)
		if err != nil {
			return err
		}
		groupInfos = append(groupInfos, groupInfo)
	}

	ret := &GroupInfoList{GroupInfos: groupInfos}
	sort.Sort(ret)
	return c.JSON(http.StatusOK, &ret)
}

func (h *Handler) ListLocalGroup(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	if groupId == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	groupIface, err := chain.GetGroupMgr().GetGroupIfaceFromIndex(groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	groupInfo, err := getGroupInfo(groupIface)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, groupInfo)
}

func (h *Handler) ListSubGroups(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	if groupId == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	subGroupIface, err := chain.GetGroupMgr().GetSubGroupIfaces(groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	var groupInfos []*GroupInfo
	for _, subGroupIface := range subGroupIface {
		groupInfo, err := getGroupInfo(subGroupIface)
		if err != nil {
			return err
		}
		groupInfos = append(groupInfos, groupInfo)
	}

	ret := &GroupInfoList{GroupInfos: groupInfos}
	sort.Sort(ret)
	return c.JSON(http.StatusOK, ret)
}

func (h *Handler) ListSubGroup(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	subGroupId = c.Param("sub_group_id")
	if groupId == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	groupIface, err := chain.GetGroupMgr().GetGroupIfaceFromIndex(groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	groupInfo, err := getGroupInfo(groupIface)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, groupInfo)
}

func getGroupInfo(groupIface def.GroupIface) (*GroupInfo, error) {

	group := &GroupInfo{}
	group.OwnerPubKey = groupIface.GetOwnerPubkey()
	group.GroupId = groupIface.GetGroupId()
	group.GroupName = groupIface.GetGroupName()
	group.ConsensusType = groupIface.GetConsensusType()
	group.AuthType = groupIface.GetAuthType()
	group.CipherKey = groupIface.GetCipherKey()
	group.AppId = groupIface.GetAppId()
	group.AppName = groupIface.GetAppName()
	group.LastUpdated = groupIface.GetLastUpdated()
	group.CurrtTopBlock = groupIface.GetCurrentBlockId()
	group.Peers = nodectx.GetNodeCtx().ListGroupPeers(groupIface.GetGroupId())
	return group, nil
}

//move to group service
/*
	b, err := base64.RawURLEncoding.DecodeString(group.UserPubkey)
	if err != nil {
		//try libp2pkey
	} else {
		ethpubkey, err := ethcrypto.DecompressPubkey(b)
		//ethpubkey, err := ethcrypto.UnmarshalPubkey(b)
		if err == nil {
			ethaddr := ethcrypto.PubkeyToAddress(*ethpubkey)
			group.UserEthaddr = ethaddr.Hex()
		}
	}
	//group.RexSyncerStatus = value.GetRexSyncerStatus()
	//group.RexSyncerResult, _ = value.ChainCtx.GetLastRexSyncResult()
*/

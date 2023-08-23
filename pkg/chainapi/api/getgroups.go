package api

import (
	"net/http"
	"sort"

	"encoding/base64"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/peer"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type GroupInfo struct {
	GroupId         string             `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	GroupName       string             `json:"group_name" validate:"required" example:"demo-app"`
	OwnerPubKey     string             `json:"owner_pubkey" validate:"required" example:"CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg=="`
	UserPubkey      string             `json:"user_pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
	UserEthaddr     string             `json:"user_eth_addr" validate:"required" example:"0495180230ae0f585ca0b4fc0767e616eaed45e400f470ed50c91668e1ed76c278b7fc5a129ff154c6b200a26cc78b7b4acc5b3915cdf66286c942aa5b65166ff5"`
	ConsensusType   string             `json:"consensus_type" validate:"required" example:"POA"`
	SyncType        string             `json:"sync_type" validate:"required" example:"PUBLIC"`
	CipherKey       string             `json:"cipher_key" validate:"required" example:"58044622d48c4d91932583a05db3ff87f29acacb62e701916f7f0bbc6e446e5d"`
	AppId           string             `json:"app_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	AppName         string             `json:"app_name" validate:"required" example:"demo-app"`
	CurrtTopBlock   uint64             `json:"currt_top_block" validate:"required" example:"0"`
	LastUpdated     int64              `json:"last_updated" validate:"required" example:"1633022375303983600"`
	RexSyncerStatus string             `json:"rex_syncer_status" validate:"required" example:"IDLE"`
	RexSyncerResult *def.RexSyncResult `json:"rex_Syncer_result" validate:"required"`
	Peers           []peer.ID          `json:"peers" validate:"required" example:"16Uiu2HAkuXLC2hZTRbWToCNztyWB39KDi8g66ou3YrSzeTbsWsFG,16Uiu2HAm8XVpfQrJYaeL7XtrHC3FvfKt2QW7P8R3MBenYyHxu8Kk"`
}

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
func (h *Handler) GetGroups(c echo.Context) (err error) {
	var groups []*GroupInfo
	groupmgr := chain.GetGroupMgr()
	for groupId, _ := range groupmgr.Groups {
		group, err := getGroupInfo(groupId)
		if err != nil {
			return err
		}
		groups = append(groups, group)
	}

	ret := GroupInfoList{groups}
	sort.Sort(&ret)
	return c.JSON(http.StatusOK, &ret)
}

// @Tags Groups
// @Summary GetGroupById
// @Description Get group info by group id
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {object} GroupInfo
// @Router /api/v1/group/{group_id} [get]
func (h *Handler) GetGroupById(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	if groupId == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}
	groupInfo, err := getGroupInfo(groupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, groupInfo)
}

func getGroupInfo(groupId string) (*GroupInfo, error) {
	groupmgr := chain.GetGroupMgr()
	value, ok := groupmgr.Groups[groupId]
	if !ok {
		return nil, rumerrors.ErrGroupNotFound
	}

	group := &GroupInfo{}

	group.OwnerPubKey = value.Item.OwnerPubKey
	group.GroupId = value.Item.GroupId
	group.GroupName = value.Item.GroupName
	group.OwnerPubKey = value.Item.OwnerPubKey
	group.UserPubkey = value.Item.UserSignPubkey
	group.ConsensusType = value.Item.ConsenseType.String()
	group.SyncType = value.Item.SyncType.String()
	group.CipherKey = value.Item.CipherKey
	group.AppId = value.Item.AppId
	group.AppName = value.Item.AppName

	group.LastUpdated = value.GetLatestUpdate()
	group.CurrtTopBlock = value.GetCurrentBlockId()

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
	group.RexSyncerStatus = value.GetRexSyncerStatus()
	group.RexSyncerResult, _ = value.ChainCtx.GetLastRexSyncResult()
	group.Peers = nodectx.GetNodeCtx().ListGroupPeers(groupId)

	return group, nil
}

package api

import (
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

type groupInfo struct {
	GroupId        string        `json:"group_id" validate:"required,uuid4"`
	GroupName      string        `json:"group_name" validate:"required"`
	OwnerPubKey    string        `json:"owner_pubkey" validate:"required"`
	UserPubkey     string        `json:"user_pubkey" validate:"required"`
	UserEthaddr    string        `json:"user_eth_addr" validate:"required"`
	ConsensusType  string        `json:"consensus_type" validate:"required"`
	EncryptionType string        `json:"encryption_type" validate:"required"`
	CipherKey      string        `json:"cipher_key" validate:"required"`
	AppKey         string        `json:"app_key" validate:"required"`
	LastUpdated    int64         `json:"last_updated" validate:"required"`
	HighestHeight  int64         `json:"highest_height" validate:"required"`
	HighestBlockId string        `json:"highest_block_id" validate:"required,uuid4"`
	GroupStatus    string        `json:"group_status" validate:"required"`
	SnapshotInfo   *snapshotInfo `json:"snapshot_info"`
}

type snapshotInfo struct {
	TimeStamp         int64
	HighestHeight     int64
	HighestBlockId    string
	Nonce             int64
	SnapshotPackageId string
	SenderPubkey      string
}

type GroupInfoList struct {
	GroupInfos []*groupInfo `json:"groups"`
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
// @Description Get all joined groups
// @Produce json
// @Success 200 {object} GroupInfoList
// @Router /api/v1/groups [get]
func (h *Handler) GetGroups(c echo.Context) (err error) {
	var groups []*groupInfo
	groupmgr := chain.GetGroupMgr()
	for _, value := range groupmgr.Groups {
		var group *groupInfo
		group = &groupInfo{}

		group.OwnerPubKey = value.Item.OwnerPubKey
		group.GroupId = value.Item.GroupId
		group.GroupName = value.Item.GroupName
		group.OwnerPubKey = value.Item.OwnerPubKey
		group.UserPubkey = value.Item.UserSignPubkey
		group.ConsensusType = value.Item.ConsenseType.String()
		group.EncryptionType = value.Item.EncryptType.String()
		group.CipherKey = value.Item.CipherKey
		group.AppKey = value.Item.AppKey
		group.LastUpdated = value.Item.LastUpdate
		group.HighestHeight = value.Item.HighestHeight
		group.HighestBlockId = value.Item.HighestBlockId

		ethaddr, err := localcrypto.Libp2pPubkeyToEthaddr(group.UserPubkey)
		if err == nil {
			group.UserEthaddr = ethaddr
		}

		switch value.GetSyncerStatus() {
		case chain.SYNCING_BACKWARD:
			group.GroupStatus = "SYNCING"
		case chain.SYNCING_FORWARD:
			group.GroupStatus = "SYNCING"
		case chain.SYNC_FAILED:
			group.GroupStatus = "SYNC_FAILED"
		case chain.IDLE:
			group.GroupStatus = "IDLE"
		}

		snapshottag, err := value.GetSnapshotInfo()
		if err != nil {
			group.SnapshotInfo = nil
		} else {
			snapshot := &snapshotInfo{}
			snapshot.TimeStamp = snapshottag.TimeStamp
			snapshot.HighestBlockId = snapshottag.HighestBlockId
			snapshot.HighestHeight = snapshottag.HighestHeight
			snapshot.Nonce = snapshottag.Nonce
			snapshot.SenderPubkey = snapshottag.SenderPubkey
			snapshot.SnapshotPackageId = snapshottag.SnapshotPackageId
			group.SnapshotInfo = snapshot
		}

		groups = append(groups, group)
	}

	ret := GroupInfoList{groups}
	sort.Sort(&ret)
	return c.JSON(http.StatusOK, &ret)
}

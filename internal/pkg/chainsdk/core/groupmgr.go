package chain

import (
	"fmt"
	"sync"
	"time"

	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var groupMgr_log = logging.Logger("groupmgr")

const JOIN_BY_API = "join_by_api"

type GroupMgrItem struct {
	ParentGroupId string
	SubGroups     map[string]*Group
}

type GroupMgr struct {
	locker        sync.Mutex
	GroupMgrItems map[string]*GroupMgrItem /*parentGroupId*/
}

var groupMgrInst *GroupMgr

func GetGroupMgr() *GroupMgr {
	return groupMgrInst
}

func InitGroupMgr() error {
	groupMgr_log.Debug("InitGroupMgr called")
	groupMgrInst = &GroupMgr{}
	groupMgrInst.GroupMgrItems = make(map[string]*GroupMgrItem)
	return nil
}

func (groupMgr *GroupMgr) LoadSubGroups(parentGroupId string) error {
	groupMgr_log.Debugf("LoadSubGroups called, parentGroupId: <%s>", parentGroupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	groupMgrItem := &GroupMgrItem{
		ParentGroupId: parentGroupId,
	}

	groupItems, err := nodectx.GetNodeCtx().GetChainStorage().GetSubGroupItems(parentGroupId)
	if err != nil {
		return err
	}

	groupMgrItem.SubGroups = make(map[string]*Group)

	for _, item := range groupItems {
		group := &Group{}
		groupMgr_log.Debugf("load group: <%s> - <%s>", parentGroupId, item.GroupId)
		groupMgrItem.SubGroups[item.GroupId] = group
		group.LoadGroup(parentGroupId, item)
		time.Sleep(1 * time.Second)
	}

	groupMgr.GroupMgrItems[parentGroupId] = groupMgrItem

	return nil
}

// load and group and start syncing
func (groupMgr *GroupMgr) StartSyncAllSubGroups(parentGroupId string) error {
	groupMgr_log.Debugf("StartSyncAllSubGroups called, parentGroupId: <%s>", parentGroupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	for _, grp := range groupMgr.GroupMgrItems[parentGroupId].SubGroups {
		groupMgr_log.Debugf("start sync group : <%s> - <%s> ", parentGroupId, grp.Item.GroupId)
		grp.StartSync()
		time.Sleep(1 * time.Second)
	}
	return nil
}

func (groupMgr *GroupMgr) StopSyncAllSubGroups(parentGroupId string) error {
	groupMgr_log.Debugf("StopSyncAllSubGroups called, parentGroupId: <%s>", parentGroupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	for _, grp := range groupMgr.GroupMgrItems[parentGroupId].SubGroups {
		groupMgr_log.Debugf("Stop sync group : <%s> - <%s>", parentGroupId, grp.Item.GroupId)
		grp.StopSync()
	}

	return nil
}

func (groupMgr *GroupMgr) TeardownAllSubGroups(parentGroupId string) {
	groupMgr_log.Debugf("TeardownAllSubGroups called, parentGroupId: <%s>", parentGroupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	for _, grp := range groupMgr.GroupMgrItems[parentGroupId].SubGroups {
		groupMgr_log.Debugf("teardown group : <%s> - <%s>", parentGroupId, grp.Item.GroupId)
		grp.Teardown()
	}
}

func (groupMgr *GroupMgr) IsGroupExist(parentGroupId, groupId string) (bool, error) {
	groupMgr_log.Debugf("IsGroupExist called, parentGroupId: <%s>, groupId: <%s>", parentGroupId, groupId)
	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.GroupMgrItems[parentGroupId]; !ok {
		return false, fmt.Errorf("parent group not exist: %s", parentGroupId)
	}

	groupMgrItem := groupMgr.GroupMgrItems[parentGroupId]
	if _, ok := groupMgrItem.SubGroups[groupId]; !ok {
		return false, fmt.Errorf("group not exist: %s", groupId)
	}

	return true, nil
}

func (groupMgr *GroupMgr) GetGroup(parentGroupId, groupId string) (*Group, error) {
	groupMgr_log.Debugf("GetGroupItem called, parentGroupId: %s, groupId: %s", parentGroupId, groupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.GroupMgrItems[parentGroupId]; !ok {
		return nil, fmt.Errorf("parent group not exist: %s", parentGroupId)
	}

	groupMgrItem := groupMgr.GroupMgrItems[parentGroupId]
	if _, ok := groupMgrItem.SubGroups[groupId]; !ok {
		return nil, fmt.Errorf("group not exist: %s", groupId)
	}

	return groupMgrItem.SubGroups[groupId], nil
}

func (groupMgr *GroupMgr) GetGroupItem(parentGroupId, groupId string) (*quorumpb.GroupItem, error) {
	groupMgr_log.Debugf("GetGroupItem called, parentGroupId: %s, groupId: %s", parentGroupId, groupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.GroupMgrItems[parentGroupId]; !ok {
		return nil, fmt.Errorf("parent group not exist: %s", parentGroupId)
	}

	groupMgrItem := groupMgr.GroupMgrItems[parentGroupId]
	if _, ok := groupMgrItem.SubGroups[groupId]; !ok {
		return nil, fmt.Errorf("group not exist: %s", groupId)
	}

	return groupMgrItem.SubGroups[groupId].Item, nil
}

func (groupMgr *GroupMgr) GetSubGroups(parentGroupid string) (map[string]*Group, error) {
	groupMgr_log.Debugf("GetSubGroupItems called, parentGroupId: %s", parentGroupid)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.GroupMgrItems[parentGroupid]; !ok {
		return nil, fmt.Errorf("parent group not exist: %s", parentGroupid)
	}

	result := make(map[string]*Group)

	for _, grp := range groupMgr.GroupMgrItems[parentGroupid].SubGroups {
		result[grp.Item.GroupId] = grp
	}

	return result, nil
}

func (groupmgr *GroupMgr) GetSubGroupIfaces(parentGroupId string) ([]chaindef.GroupIface, error) {
	groupMgr_log.Debugf("GetSubGroupIfaces called, parentGroupId: %s", parentGroupId)
	groupmgr.locker.Lock()
	defer groupmgr.locker.Unlock()

	if _, ok := groupmgr.GroupMgrItems[parentGroupId]; !ok {
		return nil, fmt.Errorf("parent group not exist: %s", parentGroupId)
	}

	var result []chaindef.GroupIface

	for _, grp := range groupmgr.GroupMgrItems[parentGroupId].SubGroups {
		result = append(result, grp)
	}

	return result, nil
}

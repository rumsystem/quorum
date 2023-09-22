package chain

import (
	"fmt"
	"sync"

	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var groupMgr_log = logging.Logger("groupmgr")

/*
	the parent of all local group is chaindef:LOCAL_GROUP
	local group item
	map [sub group id] *Group
*/

type LocalGroup struct {
	GroupId   string
	Group     *Group
	SubGroups map[string]*Group
}

type GroupMgr struct {
	locker      sync.Mutex
	LocalGroups map[string]*LocalGroup
	GroupIndex  map[string]*Group
}

var groupMgrInst *GroupMgr

func GetGroupMgr() *GroupMgr {
	return groupMgrInst
}

func InitGroupMgr() error {
	groupMgr_log.Debug("InitGroupMgr called")
	groupMgrInst = &GroupMgr{}

	groupMgrInst.LocalGroups = make(map[string]*LocalGroup)
	groupMgrInst.GroupIndex = make(map[string]*Group)

	return nil
}

func GetSubGroupIndex(parentGroupId, subGroupId string) string {
	return parentGroupId + "_" + subGroupId
}

func GetLocalGroupIndex(groupId string) string {
	return groupId
}

func (groupMgr *GroupMgr) LoadLocalGroups() error {
	groupMgr_log.Debug("LoadLocalGroups called")

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	groupItems, err := nodectx.GetNodeCtx().GetChainStorage().GetSubGroupItems(chaindef.LOCAL_GROUP)
	if err != nil {
		return err
	}
	for _, groupItem := range groupItems {
		groupMgr_log.Debugf("load local group: <%s>", groupItem.GroupId)

		group := &Group{}
		err := group.LoadGroup(chaindef.LOCAL_GROUP, groupItem)
		if err != nil {
			return err
		}

		//create localGroup and add to groupMgr
		localGroup := &LocalGroup{
			GroupId:   group.GroupId,
			Group:     group,
			SubGroups: make(map[string]*Group),
		}

		//add local group to index
		groupMgr.GroupIndex[GetLocalGroupIndex(group.GroupId)] = group

		//load all sub groups
		subGroupItesm, err := nodectx.GetNodeCtx().GetChainStorage().GetSubGroupItems(group.GroupId)
		if err != nil {
			return err
		}
		for _, subGroupItem := range subGroupItesm {
			groupMgr_log.Debugf("load sub group: <%s> - <%s>", group.GroupId, subGroupItem.GroupId)
			subGroup := &Group{}
			err := subGroup.LoadGroup(group.GroupId, subGroupItem)
			if err != nil {
				return err
			}
			localGroup.SubGroups[subGroup.GroupId] = subGroup
		}

		//add all sub groups to group index
		for _, subGroup := range localGroup.SubGroups {
			groupMgr.GroupIndex[GetSubGroupIndex(group.GroupId, subGroup.GroupId)] = subGroup
		}
	}
	return nil
}

func (groupMgr *GroupMgr) TeardownLocalGroups() {
	groupMgr_log.Debugf("TeardownLocalGroups called")

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	for _, localGroup := range groupMgr.LocalGroups {
		groupMgr_log.Debugf("teardown loacl group : <%s>", localGroup.GroupId)
		groupMgr.TeardownSubGroups(localGroup.GroupId)
		localGroup.Group.Teardown()
		//remove from group index
		delete(groupMgr.GroupIndex, GetLocalGroupIndex(localGroup.GroupId))

	}
}

func (groupMgr *GroupMgr) TeardownSubGroups(parentGroupId string) {
	groupMgr_log.Debugf("TeardownSubGroups called, parentGroupId: <%s>", parentGroupId)
	for _, subGroup := range groupMgr.LocalGroups[parentGroupId].SubGroups {
		groupMgr_log.Debugf("teardown sub group : <%s>", subGroup.GroupId)
		subGroup.Teardown()
		//remove from parent group
		delete(groupMgr.LocalGroups[parentGroupId].SubGroups, subGroup.GroupId)
		//remove from group index
		delete(groupMgr.GroupIndex, GetSubGroupIndex(parentGroupId, subGroup.GroupId))
	}
}

func (groupMgr *GroupMgr) TeardownLocalGroupById(groupId string) error {
	groupMgr_log.Debugf("TeardownLocalGroupById called, groupId: <%s>", groupId)
	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()
	if _, ok := groupMgr.LocalGroups[groupId]; !ok {
		return fmt.Errorf("local group not exist: %s", groupId)
	}

	//close all sub groups
	groupMgr.TeardownSubGroups(groupId)
	//close local group
	groupMgr.LocalGroups[groupId].Group.Teardown()

	//remove from group index
	delete(groupMgr.GroupIndex, GetLocalGroupIndex(groupId))
	//remove from local group map
	delete(groupMgr.LocalGroups, groupId)
	return nil
}

func (groupMgr *GroupMgr) TeardownSubGroupById(localGroupId, groupId string) error {
	groupMgr_log.Debugf("TeardownSubGroupById called, localGroupId: <%s>, groupId: <%s>", localGroupId, groupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[localGroupId]; !ok {
		return fmt.Errorf("local group not exist: %s", localGroupId)
	}

	if _, ok := groupMgr.LocalGroups[localGroupId].SubGroups[groupId]; !ok {
		return fmt.Errorf("sub group not exist: %s", groupId)
	}

	//close sub group
	groupMgr.LocalGroups[localGroupId].SubGroups[groupId].Teardown()
	//remove from group index
	delete(groupMgr.GroupIndex, GetSubGroupIndex(localGroupId, groupId))
	//remove from local group map
	delete(groupMgr.LocalGroups[localGroupId].SubGroups, groupId)
	return nil
}

func (groupMgr *GroupMgr) AddLocalGroup(group *Group) error {
	groupMgr_log.Debugf("AddLocalGroup called, groupId: <%s>", group.GroupId)
	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[group.GroupId]; ok {
		return fmt.Errorf("group already exist: %s", group.GroupId)
	}

	//create localGroup and add to groupMgr
	localGroup := &LocalGroup{
		GroupId:   group.GroupId,
		Group:     group,
		SubGroups: make(map[string]*Group),
	}

	//add to groupMgr
	groupMgr.LocalGroups[group.GroupId] = localGroup
	//add to group index
	groupMgr.GroupIndex[GetLocalGroupIndex(group.GroupId)] = group
	return nil
}

func (groupMgr *GroupMgr) AddSubGroup(localGroupId string, group *Group) error {
	groupMgr_log.Debugf("AddSubGroup called, parentGroupId: <%s>, groupId: <%s>", localGroupId, group.GroupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[localGroupId]; !ok {
		return fmt.Errorf("local group not exist: %s", localGroupId)
	}

	localGroup := groupMgr.LocalGroups[localGroupId]
	if _, ok := localGroup.SubGroups[group.GroupId]; ok {
		return fmt.Errorf("sub group already exist: %s", group.GroupId)
	}

	localGroup.SubGroups[group.GroupId] = group

	//check if already exist in group index
	if _, ok := groupMgr.GroupIndex[GetSubGroupIndex(localGroupId, group.GroupId)]; ok {
		groupMgr_log.Warningf("sub group already exist in other local group: %s", group.GroupId)
		return nil
	}

	//add to group index
	groupMgr.GroupIndex[GetSubGroupIndex(localGroupId, group.GroupId)] = group
	return nil
}

func (groupMgr *GroupMgr) TeamDownSubGroup(localGroupId, groupId string) error {
	groupMgr_log.Debugf("TeamDownSubGroup called, localGroupId: <%s>, groupId: <%s>", localGroupId, groupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[localGroupId]; !ok {
		return fmt.Errorf("local group not exist: %s", localGroupId)
	}

	localGroup := groupMgr.LocalGroups[localGroupId]
	if _, ok := localGroup.SubGroups[groupId]; !ok {
		return fmt.Errorf("sub group not exist: %s", groupId)
	}

	localGroup.SubGroups[groupId].Teardown()
	delete(localGroup.SubGroups, groupId)
	delete(groupMgr.GroupIndex, GetSubGroupIndex(localGroupId, groupId))
	return nil
}

func (groupMgr *GroupMgr) IsLocalGroupExist(localGroupId string) (bool, error) {
	groupMgr_log.Debugf("IsParentGroupExist called, localGroupId: <%s>", localGroupId)
	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()
	_, ok := groupMgr.LocalGroups[localGroupId]
	return ok, nil
}

func (groupMgr *GroupMgr) IsSubGroupExist(localGroupId, groupId string) (bool, error) {
	groupMgr_log.Debugf("IsSubGroupExist called, loaclGroupId: <%s>, groupId: <%s>", localGroupId, groupId)
	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[localGroupId]; !ok {
		return false, fmt.Errorf("loacl group not exist: %s", localGroupId)
	}

	localGroup := groupMgr.LocalGroups[localGroupId]
	if _, ok := localGroup.SubGroups[groupId]; !ok {
		return false, fmt.Errorf("group not exist: %s", groupId)
	}

	return true, nil
}

func (groupMgr *GroupMgr) GetGroupItem(localGroupId, groupId string) (*quorumpb.GroupItem, error) {
	groupMgr_log.Debugf("GetGroupItem called, localGroupId: %s, groupId: %s", localGroupId, groupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[localGroupId]; !ok {
		return nil, fmt.Errorf("local group not exist: %s", localGroupId)
	}

	if groupId == "" {
		return groupMgr.LocalGroups[localGroupId].Group.GroupItem, nil
	}

	localGroup := groupMgr.LocalGroups[localGroupId]
	if _, ok := localGroup.SubGroups[groupId]; !ok {
		return nil, fmt.Errorf("group not exist: %s", groupId)
	}

	return localGroup.SubGroups[groupId].GroupItem, nil
}

func (groupMgr *GroupMgr) GetSubGroupItem(localGroupId string) ([]*quorumpb.GroupItem, error) {
	groupMgr_log.Debugf("GetSubGroupItem called, localGroupId: %s", localGroupId)

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	if _, ok := groupMgr.LocalGroups[localGroupId]; !ok {
		return nil, fmt.Errorf("local group not exist: %s", localGroupId)
	}

	result := []*quorumpb.GroupItem{}
	localGroup := groupMgr.LocalGroups[localGroupId]
	for _, grp := range localGroup.SubGroups {
		result = append(result, grp.GroupItem)
	}

	return result, nil
}

func (groupMgr *GroupMgr) GetLocalGroupItems() ([]*quorumpb.GroupItem, error) {
	groupMgr_log.Debugf("GetLocalGroups called")

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	result := []*quorumpb.GroupItem{}
	for _, grp := range groupMgr.LocalGroups {
		result = append(result, grp.Group.GroupItem)
	}

	return result, nil
}

func (groupMgr *GroupMgr) GetLocalGroupIfaces() ([]chaindef.GroupIface, error) {
	groupMgr_log.Debugf("GetLocalGroups called")

	groupMgr.locker.Lock()
	defer groupMgr.locker.Unlock()

	result := []chaindef.GroupIface{}
	for _, grp := range groupMgr.LocalGroups {
		result = append(result, grp.Group)
	}

	return result, nil
}

func (groupmgr *GroupMgr) GetSubGroupIfaces(localGroupId string) ([]chaindef.GroupIface, error) {
	groupMgr_log.Debugf("GetSubGroupIface called, localGroupId: %s", localGroupId)

	groupmgr.locker.Lock()
	defer groupmgr.locker.Unlock()

	if _, ok := groupmgr.LocalGroups[localGroupId]; !ok {
		return nil, fmt.Errorf("local group not exist: %s", localGroupId)
	}

	result := []chaindef.GroupIface{}
	LocalGroup := groupmgr.LocalGroups[localGroupId]
	for _, grp := range LocalGroup.SubGroups {
		result = append(result, grp)
	}

	return result, nil
}

func (groupmgr *GroupMgr) GetGroupIface(localGroupId, subGroupId string) (chaindef.GroupIface, error) {
	groupMgr_log.Debugf("GetGroupIface called, localGroupId: %s, groupId: %s", localGroupId, subGroupId)

	groupmgr.locker.Lock()
	defer groupmgr.locker.Unlock()

	if _, ok := groupmgr.LocalGroups[localGroupId]; !ok {
		return nil, fmt.Errorf("local group not exist: %s", localGroupId)
	}

	if subGroupId == "" {
		return groupmgr.LocalGroups[localGroupId].Group, nil
	}

	LocalGroup := groupmgr.LocalGroups[localGroupId]
	if _, ok := LocalGroup.SubGroups[subGroupId]; !ok {
		return nil, fmt.Errorf("group not exist: %s", subGroupId)
	}

	return LocalGroup.SubGroups[subGroupId], nil
}

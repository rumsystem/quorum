package chain

import (
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var groupMgr_log = logging.Logger("groupmgr")

type GroupMgr struct {
	Groups map[string]*Group
}

var groupMgr *GroupMgr

func GetGroupMgr() *GroupMgr {
	return groupMgr
}

//TODO: singlaton
func InitGroupMgr() error {
	groupMgr_log.Debug("InitGroupMgr called")
	groupMgr = &GroupMgr{}
	groupMgr.Groups = make(map[string]*Group)
	return nil
}

func (groupMgr *GroupMgr) LoadAllGroups() error {
	groupMgr_log.Debug("LoadAllGroup called")
	//open all groups
	groupItemsBytes, err := nodectx.GetDbMgr().GetGroupsBytes()
	if err != nil {
		return err
	}

	for _, b := range groupItemsBytes {
		var group *Group
		group = &Group{}

		var item *quorumpb.GroupItem
		item = &quorumpb.GroupItem{}
		err := proto.Unmarshal(b, item)

		if err != nil {
			groupMgr_log.Fatalf("can't load group: %s", item.GroupId)
			groupMgr_log.Fatalf(err.Error())
		} else {
			groupMgr_log.Debugf("load group: %s", item.GroupId)
			groupMgr.Groups[item.GroupId] = group
			group.Init(item)
		}
	}
	return nil
}

//load and group and start syncing
func (groupMgr *GroupMgr) StartSyncAllGroups() error {
	groupMgr_log.Debug("SyncAllGroup called")

	for _, grp := range groupMgr.Groups {
		groupMgr_log.Debugf("Start sync group: <%s>", grp.Item.GroupId)
		grp.StartSync()
	}
	return nil
}

func (groupmgr *GroupMgr) StopSyncAllGroups() error {
	groupMgr_log.Debug("StopSyncAllGroup called")
	for _, grp := range groupMgr.Groups {
		groupMgr_log.Debugf("Stop sync group: <%s>", grp.Item.GroupId)
		go grp.StopSync()
	}
	return nil
}

func (groupmgr *GroupMgr) TeardownAllGroups() {
	groupMgr_log.Debug("Release called")
	for groupId, group := range groupmgr.Groups {
		groupMgr_log.Debugf("group: <%s> teardown", groupId)
		group.Teardown()
	}
}

func (groupmgr *GroupMgr) GetGroupItem(groupId string) (*quorumpb.GroupItem, error) {
	if grp, ok := groupmgr.Groups[groupId]; ok {
		return grp.Item, nil
	}
	return nil, fmt.Errorf("group not exist: %s", groupId)
}

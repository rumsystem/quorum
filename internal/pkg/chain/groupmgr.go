package chain

import (
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"google.golang.org/protobuf/proto"
)

type GroupMgr struct {
	dbMgr  *storage.DbMgr
	Groups map[string]*Group
}

var groupMgr *GroupMgr

var groupMgr_log = logging.Logger("groupmgr")

func GetGroupMgr() *GroupMgr {
	return groupMgr
}

//TODO: singlaton
func InitGroupMgr(dbMgr *storage.DbMgr) *GroupMgr {
	groupMgr_log.Debug("InitGroupMgr called")
	groupMgr = &GroupMgr{dbMgr: dbMgr}
	groupMgr.Groups = make(map[string]*Group)
	return groupMgr
}

//load and group and start syncing
func (groupmgr *GroupMgr) SyncAllGroup() error {
	groupMgr_log.Debug("SyncAllGroup called")

	//open all groups
	groupItemsBytes, err := groupmgr.dbMgr.GetGroupsBytes()

	if err != nil {
		return err
	}

	for _, b := range groupItemsBytes {
		var group *Group
		group = &Group{}

		var item *quorumpb.GroupItem
		item = &quorumpb.GroupItem{}

		proto.Unmarshal(b, item)
		group.Init(item)
		if err == nil {
			groupMgr_log.Debugf("Start sync group: %s", item.GroupId)
			go group.StartSync()
			groupmgr.Groups[item.GroupId] = group
		} else {
			groupMgr_log.Fatalf("can't sync group: %s", item.GroupId)
			groupMgr_log.Fatalf(err.Error())
		}
	}

	return nil
}

func (groupmgr *GroupMgr) StopSyncAllGroup() error {
	groupMgr_log.Debug("StopSyncAllGroup called")
	return nil
}

func (groupmgr *GroupMgr) Release() {
	groupMgr_log.Debug("Release called")
	for groupId, group := range groupmgr.Groups {
		groupMgr_log.Debugf("group: <%s> teardown", groupId)
		group.Teardown()
	}
	//close ctx db
	groupmgr.dbMgr.CloseDb()
}

func (groupmgr *GroupMgr) GetGroupItem(groupId string) (*quorumpb.GroupItem, error) {
	if grp, ok := groupmgr.Groups[groupId]; ok {
		return grp.Item, nil
	}
	return nil, fmt.Errorf("group not exist: %s", groupId)

}

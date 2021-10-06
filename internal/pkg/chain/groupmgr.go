package chain

import (
	"fmt"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/huo-ju/quorum/internal/pkg/storage"
	"google.golang.org/protobuf/proto"
)

type GroupMgr struct {
	dbMgr  *storage.DbMgr
	Groups map[string]*Group
}

var groupMgr *GroupMgr

func GetGroupMgr() *GroupMgr {
	return groupMgr
}

//TODO: singlaton
func InitGroupMgr(dbMgr *storage.DbMgr) *GroupMgr {
	groupMgr = &GroupMgr{dbMgr: dbMgr}
	groupMgr.Groups = make(map[string]*Group)
	return groupMgr
}

//load and group and start syncing
func (groupmgr *GroupMgr) SyncAllGroup() error {
	chain_log.Infof("Start Sync all groups")

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
			chain_log.Infof(fmt.Sprintf("Start sync group: %s", item.GroupId))
			go group.StartSync()
			groupmgr.Groups[item.GroupId] = group
		} else {
			chain_log.Infof(fmt.Sprintf("can't init group: %s", item.GroupId))
			chain_log.Fatalf(err.Error())
		}
	}

	return nil
}

func (groupmgr *GroupMgr) StopSyncAllGroup() error {
	return nil
}

func (groupmgr *GroupMgr) Release() {
	//close all groups
	for groupId, group := range groupmgr.Groups {
		fmt.Println("group:", groupId, " teardown")
		group.Teardown()
	}
	//close ctx db
	groupmgr.dbMgr.CloseDb()
}

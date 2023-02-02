package appdata

import (
	"strconv"
	"time"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var appsynclog = logging.Logger("appsync")

type AppSync struct {
	appdb    *AppDb
	dbmgr    *storage.DbMgr
	groupmgr *chain.GroupMgr
	apiroot  string
	nodename string
}

func NewAppSyncAgent(apiroot string, nodename string, appdb *AppDb, dbmgr *storage.DbMgr) *AppSync {
	groupmgr := chain.GetGroupMgr()
	appsync := &AppSync{appdb, dbmgr, groupmgr, apiroot, nodename}
	return appsync
}

func (appsync *AppSync) GetGroups() []*quorumpb.GroupItem {
	var items []*quorumpb.GroupItem
	for _, grp := range appsync.groupmgr.Groups {
		items = append(items, grp.Item)
	}
	return items
}

func (appsync *AppSync) ParseBlockTrxs(groupid string, block *quorumpb.Block) error {
	appsynclog.Infof("ParseBlockTrxs %d trx(s) on group %s epoch %d", len(block.Trxs), groupid, block.Epoch)
	err := appsync.appdb.AddMetaByTrx(block.Epoch, groupid, block.Trxs)
	if err != nil {
		appsynclog.Errorf("ParseBlockTrxs on group %s err:  ", groupid, err)
	}
	return err
}

func (appsync *AppSync) RunSync(groupid string, lastSyncEpoch int64, highestepoch int64) {

	for {
		if lastSyncEpoch >= highestepoch {
			break
		}
		lastSyncEpoch++
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(groupid, lastSyncEpoch, false, appsync.nodename)
		if err == nil {
			err := appsync.ParseBlockTrxs(groupid, block)
			if err != nil {
				appsynclog.Errorf("<%s> epoch %d ParseBlockTrxs error %s", groupid, block.Epoch, err)
				break
			}

		} else {
			appsynclog.Errorf("db read err: %s, groupid: %s, lastSyncEpoch : %d, HighestEpoch: %d", err, groupid, lastSyncEpoch, highestepoch)
			break
		}
	}
}

func (appsync *AppSync) Start(interval int) {
	go func() {
		for {
			groups := appsync.GetGroups()
			for _, groupitem := range groups {
				groupId := groupitem.GroupId
				group, ok := appsync.groupmgr.Groups[groupId]
				if !ok {
					appsynclog.Errorf("can not find group : %s", groupId)
					continue
				}

				epochstr, err := appsync.appdb.GetGroupStatus(groupId, "Epoch")
				if err == nil {
					if epochstr == "" { //init, set to 0
						epochstr = "0"
					}
				} else {
					appsynclog.Errorf("sync group : %s GetGroupStatus err %s", groupId, err)
					continue
				}

				lastSyncEpoch, err := strconv.ParseInt(epochstr, 10, 64)
				if err == nil {
					if group.GetCurrentEpoch() > lastSyncEpoch {
						appsync.RunSync(groupId, lastSyncEpoch, group.GetCurrentEpoch())
					}
				} else {
					appsynclog.Errorf("sync group : %s Get Group last sync Epoch err %s", groupId, err)
				}
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}
	}()
}

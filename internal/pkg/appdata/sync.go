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
	appsynclog.Infof("ParseBlockTrxs %d trx(s) on group %s blockId <%d>", len(block.Trxs), groupid, block.BlockId)
	err := appsync.appdb.AddMetaByTrx(block.BlockId, groupid, block.Trxs)
	if err != nil {
		appsynclog.Errorf("ParseBlockTrxs on group %s err:  ", groupid, err)
	}
	return err
}

func (appsync *AppSync) RunSync(groupid string, lastSyncBlock uint64, highestBlock uint64) {

	for {
		if lastSyncBlock >= highestBlock {
			break
		}
		lastSyncBlock++
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(groupid, lastSyncBlock, false, appsync.nodename)
		if err == nil {
			err := appsync.ParseBlockTrxs(groupid, block)
			if err != nil {
				appsynclog.Errorf("<%s> epoch %d ParseBlockTrxs error %s", groupid, block.Epoch, err)
				break
			}

		} else {
			appsynclog.Errorf("db read err: %s, groupid: %s, lastSyncEpoch : %d, HighestEpoch: %d", err, groupid, lastSyncBlock, highestBlock)
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

				blockIdStr, err := appsync.appdb.GetGroupStatus(groupId, "BlockId")
				if err == nil {
					if blockIdStr == "" { //init, set to 0
						blockIdStr = "0"
					}
				} else {
					appsynclog.Errorf("sync group : %s GetGroupStatus err %s", groupId, err)
					continue
				}

				lastSyncBlock, err := strconv.ParseUint(blockIdStr, 10, 64)
				if err == nil {
					if group.GetCurrentBlockId() > lastSyncBlock {
						appsync.RunSync(groupId, lastSyncBlock, group.GetCurrentBlockId())
					}
				} else {
					appsynclog.Errorf("sync group : %s Get Group last sync block err %s", groupId, err)
				}
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}
	}()
}

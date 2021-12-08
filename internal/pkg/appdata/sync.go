package appdata

import (
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/pkg/wasm/logger"
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

/*
func highestBlockIdToStr(HighestBlockId []string) string {
	if len(HighestBlockId) > 1 {
		sort.Strings(HighestBlockId)
		return strings.Join(HighestBlockId, "_")
	} else if len(HighestBlockId) == 1 {
		return HighestBlockId[0]
	}
	return ""
}
*/
func (appsync *AppSync) ParseBlockTrxs(groupid string, block *quorumpb.Block) ([]*quorumpb.Block, error) {
	appsynclog.Infof("ParseBlockTrxs %d trx(s) on group %s", len(block.Trxs), groupid)
	err := appsync.appdb.AddMetaByTrx(block.BlockId, groupid, block.Trxs)
	if err != nil {
		appsynclog.Errorf("ParseBlockTrxs on group %s err:  ", groupid, err)
	}
	return appsync.dbmgr.GetSubBlock(block.BlockId, appsync.nodename)
}

func (appsync *AppSync) RunSync(groupid string, lastBlockId string, newBlockId string) {
	var blocks []*quorumpb.Block
	subblocks, err := appsync.dbmgr.GetSubBlock(lastBlockId, appsync.nodename)
	if err == nil {
		blocks = append(blocks, subblocks...)
		for {
			if len(blocks) == 0 {
				appsynclog.Infof("no new blocks, skip sync.")
				break
			}

			var blk *quorumpb.Block
			blk, blocks = blocks[0], blocks[1:]
			newsubblocks, err := appsync.ParseBlockTrxs(groupid, blk)
			if err == nil {
				blocks = append(blocks, newsubblocks...)
			} else {
				appsynclog.Errorf("ParseBlockTrxs error %s", err)
			}

		}
	} else {
		appsynclog.Errorf("db read err: %s, lastBlockId: %s, newBlockId: %s", err, lastBlockId, newBlockId)
	}
}

func (appsync *AppSync) Start(interval int) {
	go func() {
		for {
			groups := appsync.GetGroups()
			for _, groupitem := range groups {
				lastBlockId, err := appsync.appdb.GetGroupStatus(groupitem.GroupId, "HighestBlockId")
				if err == nil {
					if lastBlockId == "" {
						lastBlockId = groupitem.GenesisBlock.BlockId
					}
					//newBlockid := highestBlockIdToStr(groupitem.HighestBlockId)
					if lastBlockId != groupitem.HighestBlockId {
						logger.Console.Debug("lastBlockId: " + lastBlockId)
						appsync.RunSync(groupitem.GroupId, lastBlockId, groupitem.HighestBlockId)
					}
				} else {
					appsynclog.Errorf("sync group : %s Get HighestBlockId err %s", groupitem.GroupId, err)
				}
			}
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}()
}

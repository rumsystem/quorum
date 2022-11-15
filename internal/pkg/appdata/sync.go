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
	appsynclog.Infof("ParseBlockTrxs %d trx(s) on group %s", len(block.Trxs), groupid)
	err := appsync.appdb.AddMetaByTrx(block.Epoch, groupid, block.Trxs)
	if err != nil {
		appsynclog.Errorf("ParseBlockTrxs on group %s err:  ", groupid, err)
	}
	return err
}

//return the length of the path between the from and to block. and if the path can reach to the toblock.
func (appsync *AppSync) chainLength(fromBlockId string, toBlockId string) (uint, bool) {
	/*
		var chainlength uint
		if fromBlockId == toBlockId {
			return 0, true //reach the toblock
		}
		nextblockid := fromBlockId

		for {
			subblocks, err := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(nextblockid, appsync.nodename)
			if err != nil {
				return 0, false //error
			}
			if len(subblocks) == 0 {
				return chainlength, false //can not reach the toblock
			} else if len(subblocks) == 1 {
				chainlength += 1
				nextblockid = subblocks[0].BlockId
			} else if len(subblocks) > 1 { //multi path, calculate every paths
				var subchainlen uint
				nextsubblkid := ""
				for _, blk := range subblocks {
					l, s := appsync.chainLength(blk.BlockId, toBlockId)
					if l > subchainlen && s == true { //find a longer chain and can reach the toBlock
						subchainlen = l
						nextsubblkid = blk.BlockId
					}
				}
				nextblockid = nextsubblkid
			}
		}*/

	//added by cuicat
	return 0, false
}

func (appsync *AppSync) findNextBlock(blocks []*quorumpb.Block, toBlockId string) *quorumpb.Block {
	/*
		var nextsubblk *quorumpb.Block
		var subchainlen uint
		if len(blocks) == 1 {
			nextsubblk = blocks[0]
		} else {
			for _, blk := range blocks {
				l, s := appsync.chainLength(blk.BlockId, toBlockId)
				if l > subchainlen && s == true { //reach the toblock
					subchainlen = l
					nextsubblk = blk
				}
			}
		}

		return nextsubblk
	*/

	return nil
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
				appsynclog.Errorf("ParseBlockTrxs error %s", err)
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

				epochstr, err := appsync.appdb.GetGroupStatus(groupitem.GroupId, "Epoch")
				if err == nil {
					if epochstr == "" { //init, set to 0
						epochstr = "0"
					}
				} else {
					appsynclog.Errorf("sync group : %s GetGroupStatus err %s", groupitem.GroupId, err)
					break
				}
				lastSyncEpoch, err := strconv.ParseInt(epochstr, 10, 64)
				if err == nil {
					if groupitem.Epoch > lastSyncEpoch {
						appsync.RunSync(groupitem.GroupId, lastSyncEpoch, groupitem.Epoch)
					}
				} else {
					appsynclog.Errorf("sync group : %s Get Group last sync Epoch err %s", groupitem.GroupId, err)
				}
			}
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}()
}

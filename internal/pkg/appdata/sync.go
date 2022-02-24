package appdata

import (
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
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

func (appsync *AppSync) ParseBlockTrxs(groupid string, block *quorumpb.Block) ([]*quorumpb.Block, error) {
	appsynclog.Infof("ParseBlockTrxs %d trx(s) on group %s", len(block.Trxs), groupid)
	err := appsync.appdb.AddMetaByTrx(block.BlockId, groupid, block.Trxs)
	if err != nil {
		appsynclog.Errorf("ParseBlockTrxs on group %s err:  ", groupid, err)
	}
	return appsync.dbmgr.GetSubBlock(block.BlockId, appsync.nodename)
}

//return the length of the path between the from and to block. and if the path can reach to the toblock.
func (appsync *AppSync) chainLength(fromBlockId string, toBlockId string) (uint, bool) {
	var chainlength uint
	if fromBlockId == toBlockId {
		return 0, true //reach the toblock
	}
	nextblockid := fromBlockId

	for {
		subblocks, err := appsync.dbmgr.GetSubBlock(nextblockid, appsync.nodename)
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
	}
}

func (appsync *AppSync) findNextBlock(blocks []*quorumpb.Block, toBlockId string) *quorumpb.Block {
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
}

func (appsync *AppSync) RunSync(groupid string, lastBlockId string, newBlockId string) {
	var nextblock *quorumpb.Block
	subblocks, err := appsync.dbmgr.GetSubBlock(lastBlockId, appsync.nodename)
	if err == nil {
		nextblock = appsync.findNextBlock(subblocks, newBlockId)
		for {
			if nextblock == nil {
				appsynclog.Infof("no new blocks, skip sync.")
				break
			}
			newsubblocks, err := appsync.ParseBlockTrxs(groupid, nextblock)
			if err == nil {
				nextblock = appsync.findNextBlock(newsubblocks, newBlockId)
			} else {
				appsynclog.Errorf("ParseBlockTrxs error %s", err)
			}

		}
	} else {
		appsynclog.Errorf("db read err: %s, groupid: %s, lastBlockId: %s, newBlockId: %s", err, groupid, lastBlockId, newBlockId)
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
					if lastBlockId != groupitem.HighestBlockId {
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

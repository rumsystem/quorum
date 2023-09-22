package appdata

import (
	"strconv"
	"sync"
	"time"

	"github.com/edwingeng/deque/v2"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var (
	appsynclog = logging.Logger("appsync")

	once                     sync.Once
	onChainTrxQueue          *deque.Deque[*OnChainTrxEvent]
	maxOnChainTrxQueueLength = 2000
)

type OnChainTrxEvent struct {
	GroupId string `json:"group_id"`
	TrxId   string `json:"trx_id"`
}

type AppSync struct {
	appdb    *AppDb
	dbmgr    *storage.DbMgr
	groupmgr *chain.GroupMgr
	apiroot  string
	nodename string
}

func GetOnChainTrxQueue() *deque.Deque[*OnChainTrxEvent] {
	once.Do(func() {
		onChainTrxQueue = deque.NewDeque[*OnChainTrxEvent]()
	})

	return onChainTrxQueue
}

func pushOnChainTrxQueue(trxs []*quorumpb.Trx) {
	q := GetOnChainTrxQueue()
	for _, trx := range trxs {
		item := OnChainTrxEvent{
			GroupId: trx.GroupId,
			TrxId:   trx.TrxId,
		}
		appsynclog.Debugf("put on chain trx event: %+v to queue", item)
		if q.Len() >= maxOnChainTrxQueueLength {
			q.Clear()
		}
		q.PushFront(&item)
	}
}

func NewAppSyncAgent(apiroot string, nodename string, appdb *AppDb, dbmgr *storage.DbMgr) *AppSync {
	groupmgr := chain.GetGroupMgr()
	appsync := &AppSync{appdb, dbmgr, groupmgr, apiroot, nodename}
	return appsync
}

func (appsync *AppSync) ParseBlockTrxs(groupid string, block *quorumpb.Block) error {
	appsynclog.Infof("ParseBlockTrxs %d trx(s) on group %s blockId <%d>", len(block.Trxs), groupid, block.BlockId)
	err := appsync.appdb.AddMetaByTrx(block.BlockId, groupid, block.Trxs)
	if err != nil {
		appsynclog.Errorf("ParseBlockTrxs on group %s err:  ", groupid, err)
		return err
	}

	pushOnChainTrxQueue(block.Trxs)

	return nil
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
				appsynclog.Errorf("<%s> BlockId %d ParseBlockTrxs error %s", groupid, block.BlockId, err)
				break
			}

		} else {
			appsynclog.Errorf("db read err: %s, groupid: %s, lastSyncEpoch : %d, HighestEpoch: %d", err, groupid, lastSyncBlock, highestBlock)
			break
		}
	}
}

func (appsync *AppSync) StartSyncLocalGroups(interval int) {
	go func() {
		for {
			groupIfaces, err := appsync.groupmgr.GetLocalGroupIfaces()
			if err != nil {
				appsynclog.Debugf("get local groups err %s", err)
				return
			}

			for _, iface := range groupIfaces {
				groupId := iface.GetGroupId()
				blockIdStr, err := appsync.appdb.GetGroupStatus(groupId, "Block")
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
					currBlkId := iface.GetCurrentBlockId()
					if currBlkId > lastSyncBlock {
						appsync.RunSync(groupId, lastSyncBlock, currBlkId)
					}
				} else {
					appsynclog.Errorf("sync group : %s Get Group last sync block err %s", groupId, err)
				}
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}
	}()
}

package chain

import (
	"encoding/binary"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var snapshot_log = logging.Logger("snapshot")

const DEFAULT_SNAPSHOT_INTERVAL time.Duration = 60 //60s
const PACKAGE_TOTAL_SIZE int = 900 * 1024          //Maximum package payload is 900K

type SnapshotStatus int

const (
	SnapShotIdle SnapshotStatus = iota
	SnapShotRunning
)

type MolassesSnapshot struct {
	grpItem  *quorumpb.GroupItem
	cIface   ChainMolassesIface
	nodename string
	ticker   *time.Ticker
	groupId  string
	status   SnapshotStatus
	statusmu sync.RWMutex
	stopChan chan bool
}

func (snapshot *MolassesSnapshot) Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface) {
	snapshot_log.Debug("Init called")
	snapshot.grpItem = item
	snapshot.cIface = iface
	snapshot.nodename = nodename
	snapshot.groupId = item.GroupId
	snapshot.status = SnapShotIdle

	snapshot_log.Infof("<%s> producer created", snapshot.groupId)
}

func (snapshot *MolassesSnapshot) SetInterval(sec int) {
	snapshot_log.Debug("SetInterval called")
}

func (snapshot *MolassesSnapshot) Start() error {
	snapshot_log.Debug("Start called")

	if snapshot.status == SnapShotIdle {
		snapshot_log.Warningf("<%s> Snapshot already started", snapshot.groupId)
		return nil
	}

	snapshot.statusmu.Lock()
	snapshot.status = SnapShotRunning
	snapshot.statusmu.Unlock()

	snapshot.stopChan = snapshot.startTicker()
	return nil
}

func (snapshot *MolassesSnapshot) startTicker() chan bool {
	snapshot_log.Debug("startTicker called")
	snapshot.ticker = time.NewTicker(DEFAULT_SNAPSHOT_INTERVAL * time.Second)
	stopChan := make(chan bool)

	go func(ticker *time.Ticker) {
		defer ticker.Stop()
		for {
			select {
			case <-snapshot.ticker.C:
				snapshot.SendSnapshot()
			case stop := <-stopChan:
				if stop {
					snapshot_log.Debug("ticker stopped by signal")
					return
				}
			}
		}

	}(snapshot.ticker)
	return stopChan

}

func (snapshot *MolassesSnapshot) Stop() error {
	snapshot_log.Debug("Stop called")
	close(snapshot.stopChan)
	return nil
}

func (snapshot *MolassesSnapshot) SendSnapshot() error {
	snapshot_log.Debugf("<%s> sendSnapshot called", snapshot.groupId)
	snapshotItems, err := snapshot.getSnapshotItem()

	if err != nil {
		return err
	}

	//if len(snapshotItems) == 0 {
	//	return nil
	//}

	//send snapshot anyway, even if no snapshot item found
	var snapshotsToSend []*quorumpb.Snapshot
	for len(snapshotItems) != 0 {
		var snapshotpackage *quorumpb.Snapshot
		snapshotpackage = &quorumpb.Snapshot{}
		snapshotpackage.SnapshotId = guuid.NewString()
		snapshotpackage.GroupId = snapshot.groupId
		nonce, err := nodectx.GetDbMgr().GetNextNouce(snapshot.groupId, snapshot.nodename)
		if err != nil {
			return err
		}
		snapshotpackage.Nonce = int64(nonce)
		snapshotpackage.SenderPubkey = snapshot.grpItem.OwnerPubKey
		snapshotpackage.TimeStamp = time.Now().UnixNano()
		snapshotpackage.HighestHeight = snapshot.grpItem.HighestHeight
		snapshotpackage.HighestBlockId = snapshot.grpItem.HighestBlockId

		sizeCount := 0
		itemCount := 0
		for _, snapshotItem := range snapshotItems {
			for sizeCount < PACKAGE_TOTAL_SIZE {
				encodedContent, _ := quorumpb.ContentToBytes(snapshotItem)
				snapshotpackage.SnapshotItems = append(snapshotpackage.SnapshotItems, snapshotItem)
				sizeCount += binary.Size(encodedContent)
				itemCount++
			}
			break
		}

		//remove packaged snapshotItem from list
		snapshotItems = snapshotItems[itemCount:]
		snapshotsToSend = append(snapshotsToSend, snapshotpackage)
	}

	connMgr, err := conn.GetConn().GetConnMgr(snapshot.groupId)
	if err != nil {
		return err
	}

	snapshotPackageId := guuid.NewString()
	for _, item := range snapshotsToSend {
		//for a bundle of snapshots, use same snapshotpackageId
		item.SnapshotPackageId = snapshotPackageId

		//for this round snapshot,how many data packages are included
		item.TotalCount = int64(len(snapshotItems))
		bbytes, err := proto.Marshal(item)
		if err != nil {
			return err
		}
		hash := Hash(bbytes)
		item.Hash = hash

		signature, err := nodectx.GetNodeCtx().Keystore.SignByKeyName(snapshot.groupId, hash, snapshot.nodename)
		if err != nil {
			return err
		}
		item.Singature = signature

		//send it out
		connMgr.SendSnapshotPsconn(item, conn.UserChannel)
	}

	return nil
}

func (snapshot *MolassesSnapshot) getSnapshotItem() ([]*quorumpb.SnapshotItem, error) {
	snapshot_log.Debugf("<%s> GetSnapshot called", snapshot.groupId)

	var result []*quorumpb.SnapshotItem
	var appConfigs [][]byte
	appConfigs, err := nodectx.GetDbMgr().GetAllAppConfigInBytes(snapshot.groupId, snapshot.nodename)
	if err != nil {
		return nil, err
	}
	for _, appConfig := range appConfigs {
		var snapshotItem *quorumpb.SnapshotItem
		snapshotItem = &quorumpb.SnapshotItem{}
		snapshotItem.SnapshotItemId = guuid.New().String()
		snapshotItem.Type = quorumpb.SnapShotItemType_SNAPSHOT_APP_CONFIG
		snapshotItem.Data = appConfig
		result = append(result, snapshotItem)
	}

	var chainConfig [][]byte
	chainConfig, err = nodectx.GetDbMgr().GetAllChainConfigInBytes(snapshot.groupId, snapshot.nodename)
	if err != nil {
		return nil, err
	}
	for _, chainConfig := range chainConfig {
		var snapshotItem *quorumpb.SnapshotItem
		snapshotItem = &quorumpb.SnapshotItem{}
		snapshotItem.SnapshotItemId = guuid.New().String()
		snapshotItem.Type = quorumpb.SnapShotItemType_SNAPSHOT_CHAIN_CONFIG
		snapshotItem.Data = chainConfig
		result = append(result, snapshotItem)
	}

	return result, nil
}

func (snapshot *MolassesSnapshot) ApplySnapshot() error {
	return nil
}

package chain

import (
	"encoding/binary"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var snapshotsender_log = logging.Logger("sssender")

const DEFAULT_SNAPSHOT_INTERVAL time.Duration = 60 //60s
const PACKAGE_TOTAL_SIZE int = 900 * 1024          //Maximum package payload is 900K

type SnapshotSenderStatus int

const (
	SENDER_RUNNING SnapshotSenderStatus = iota
	SENDER_IDLE
)

/*
	ADD DESCRIPTION for SNAPSHOTPACKAGE HERE
*/

type MolassesSnapshotSender struct {
	grpItem      *quorumpb.GroupItem
	cIface       ChainMolassesIface
	nodename     string
	ticker       *time.Ticker
	groupId      string
	status       SnapshotSenderStatus
	statusmu     sync.RWMutex
	stopChan     chan bool
	lastestNonce int64
}

func (sssender *MolassesSnapshotSender) Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface) {
	snapshotsender_log.Debugf("<%s> Init called", sssender.groupId)
	sssender.grpItem = item
	sssender.cIface = iface
	sssender.nodename = nodename
	sssender.groupId = item.GroupId
	sssender.status = SENDER_IDLE
}

func (sssender *MolassesSnapshotSender) SetInterval(sec int) {
	snapshotsender_log.Debugf("<%s> SetInterval called", sssender.groupId)
}

func (sssender *MolassesSnapshotSender) Start() error {
	snapshotsender_log.Debugf("<%s> Start called", sssender.groupId)

	if sssender.status == SENDER_RUNNING {
		snapshotsender_log.Debugf("<%s> sender running, no need to start again", sssender.groupId)
		return nil
	}

	sssender.statusmu.Lock()
	sssender.status = SENDER_RUNNING
	sssender.statusmu.Unlock()

	sssender.stopChan = sssender.startTicker()
	return nil
}

func (sssender *MolassesSnapshotSender) startTicker() chan bool {
	snapshotsender_log.Debugf("<%s> startTicker called", sssender.groupId)
	sssender.ticker = time.NewTicker(DEFAULT_SNAPSHOT_INTERVAL * time.Second)
	stopChan := make(chan bool)

	go func(ticker *time.Ticker) {
		defer ticker.Stop()
		for {
			select {
			case <-sssender.ticker.C:
				sssender.sendSnapshot()
			case stop := <-stopChan:
				if stop {
					snapshotsender_log.Debugf("<%s>ticker stopped by signal", sssender.groupId)
					return
				}
			}
		}

	}(sssender.ticker)
	return stopChan
}

func (sssender *MolassesSnapshotSender) Stop() error {
	snapshotsender_log.Debugf("<%s> Stop called", sssender.groupId)

	if sssender.status == SENDER_IDLE {
		snapshotsender_log.Debugf("<%s> sender idle, no need to stop", sssender.groupId)
		return nil
	}

	close(sssender.stopChan)

	sssender.statusmu.Lock()
	sssender.status = SENDER_IDLE
	sssender.statusmu.Unlock()

	return nil
}

func (sssender *MolassesSnapshotSender) sendSnapshot() error {
	snapshotsender_log.Debugf("<%s> sendSnapshot called", sssender.groupId)
	snapshotItems, err := sssender.getSnapshotItems()
	if err != nil {
		return err
	}

	var snapshotsToSend []*quorumpb.Snapshot
	if len(snapshotItems) == 0 {
		snapshotsender_log.Debugf("<%s> create empty Snapshot", sssender.groupId)
		var snapshotpackage *quorumpb.Snapshot
		snapshotpackage = &quorumpb.Snapshot{}
		snapshotpackage.SnapshotId = guuid.NewString()
		snapshotpackage.GroupId = sssender.groupId
		nonce, err := nodectx.GetDbMgr().GetNextNouce(sssender.groupId, sssender.nodename)
		if err != nil {
			return err
		}
		snapshotpackage.Nonce = int64(nonce)
		snapshotpackage.SenderPubkey = sssender.grpItem.OwnerPubKey
		snapshotpackage.TimeStamp = time.Now().UnixNano()
		snapshotpackage.HighestHeight = sssender.grpItem.HighestHeight
		snapshotpackage.HighestBlockId = sssender.grpItem.HighestBlockId
		snapshotsToSend = append(snapshotsToSend, snapshotpackage)
	} else {
		snapshotsender_log.Debugf("<%s> create Snapshot with configuration item", sssender.groupId)
		for len(snapshotItems) != 0 {
			var snapshotpackage *quorumpb.Snapshot
			snapshotpackage = &quorumpb.Snapshot{}
			snapshotpackage.SnapshotId = guuid.NewString()
			snapshotpackage.GroupId = sssender.groupId
			snapshotpackage.SenderPubkey = sssender.grpItem.OwnerPubKey
			snapshotpackage.HighestHeight = sssender.grpItem.HighestHeight
			snapshotpackage.HighestBlockId = sssender.grpItem.HighestBlockId

			sizeCount := 0
			itemCount := 0

			for _, snapshotItem := range snapshotItems {
				if sizeCount < PACKAGE_TOTAL_SIZE {
					encodedContent, _ := quorumpb.ContentToBytes(snapshotItem)
					snapshotpackage.SnapshotItems = append(snapshotpackage.SnapshotItems, snapshotItem)
					sizeCount += binary.Size(encodedContent)
					itemCount++
					//if all items packaged
					if itemCount == len(snapshotItems) {
						break
					}
				} else {
					//package is full, break and create a new one
					break
				}
			}

			//remove packaged snapshotItem from list
			snapshotItems = snapshotItems[itemCount:]
			snapshotsToSend = append(snapshotsToSend, snapshotpackage)
		}
	}

	connMgr, err := conn.GetConn().GetConnMgr(sssender.groupId)
	if err != nil {
		return err
	}

	snapshotPackageId := guuid.NewString()
	for _, item := range snapshotsToSend {
		//for a bundle of snapshots, use same snapshotpackageId
		item.SnapshotPackageId = snapshotPackageId

		//for a bundle of snapshots, use same nonce
		nonce, err := nodectx.GetDbMgr().GetNextNouce(sssender.groupId, sssender.nodename)
		if err != nil {
			return err
		}
		item.Nonce = int64(nonce)

		//for a bundle of snapshots, user same timestamp
		item.TimeStamp = time.Now().UnixNano()

		//for this round of snapshot, how many data packages are included
		item.TotalCount = int64(len(snapshotsToSend))
		bbytes, err := proto.Marshal(item)
		if err != nil {
			return err
		}
		hashed := localcrypto.Hash(bbytes)
		signature, err := nodectx.GetNodeCtx().Keystore.SignByKeyName(sssender.groupId, hashed, sssender.nodename)
		if err != nil {
			return err
		}

		item.Singature = signature
		//send it out
		connMgr.SendSnapshotPsconn(item, conn.UserChannel)
	}

	snapshotsender_log.Debugf("<%s> Snapshot sent, total count <%d>", sssender.groupId, len(snapshotsToSend))
	return nil
}

func (sssender *MolassesSnapshotSender) getSnapshotItems() ([]*quorumpb.SnapshotItem, error) {
	snapshotsender_log.Debugf("<%s> getSnapshotItems called", sssender.groupId)

	var result []*quorumpb.SnapshotItem
	var appConfigs [][]byte
	appConfigs, err := nodectx.GetDbMgr().GetAllAppConfigInBytes(sssender.groupId, sssender.nodename)
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
	chainConfig, err = nodectx.GetDbMgr().GetAllChainConfigInBytes(sssender.groupId, sssender.nodename)
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

	var users [][]byte
	users, err = nodectx.GetDbMgr().GetAllUserInBytes(sssender.groupId, sssender.nodename)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		var snapshotItem *quorumpb.SnapshotItem
		snapshotItem = &quorumpb.SnapshotItem{}
		snapshotItem.SnapshotItemId = guuid.New().String()
		snapshotItem.Type = quorumpb.SnapShotItemType_SNAPSHOT_USER
		snapshotItem.Data = user
		result = append(result, snapshotItem)
	}

	var producers [][]byte
	users, err = nodectx.GetDbMgr().GetAllProducerInBytes(sssender.groupId, sssender.nodename)
	if err != nil {
		return nil, err
	}
	for _, producer := range producers {
		var snapshotItem *quorumpb.SnapshotItem
		snapshotItem = &quorumpb.SnapshotItem{}
		snapshotItem.SnapshotItemId = guuid.New().String()
		snapshotItem.Type = quorumpb.SnapShotItemType_SNAPSHOT_PRODUCER
		snapshotItem.Data = producer
		result = append(result, snapshotItem)
	}

	var announces [][]byte
	announces, err = nodectx.GetDbMgr().GetAllAnnounceInBytes(sssender.groupId, sssender.nodename)
	if err != nil {
		return nil, err
	}
	for _, announce := range announces {
		var snapshotItem *quorumpb.SnapshotItem
		snapshotItem = &quorumpb.SnapshotItem{}
		snapshotItem.SnapshotItemId = guuid.New().String()
		snapshotItem.Type = quorumpb.SnapShotItemType_SNAPSHOT_ANNOUNCE
		snapshotItem.Data = announce
		result = append(result, snapshotItem)
	}

	return result, nil
}

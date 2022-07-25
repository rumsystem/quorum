package chain

import (
	"encoding/base64"
	"errors"
	"sync"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var snapshotreceiver_log = logging.Logger("ssreceiver")

type MolassesSnapshotReceiver struct {
	grpItem           *quorumpb.GroupItem
	nodename          string
	groupId           string
	snapshotpackages  map[string](map[string]*quorumpb.Snapshot)
	latestNonce       int64
	latestBlockId     string
	latestBlockHeight int64
	snapshotTag       *quorumpb.SnapShotTag
	applySnapshotMu   sync.RWMutex
}

func (ssreceiver *MolassesSnapshotReceiver) Init(item *quorumpb.GroupItem, nodename string) {
	snapshotreceiver_log.Debugf("<%s> Init called", item.GroupId)
	ssreceiver.grpItem = item
	ssreceiver.nodename = nodename
	ssreceiver.groupId = item.GroupId
	ssreceiver.snapshotpackages = make(map[string]map[string]*quorumpb.Snapshot)
	snapshotTag, err := nodectx.GetNodeCtx().GetChainStorage().GetSnapshotTag(item.GroupId, nodename)
	if err != nil {
		snapshotTag = &quorumpb.SnapShotTag{}
		snapshotTag.Nonce = 0
		snapshotTag.HighestBlockId = ""
		snapshotTag.TimeStamp = 0
		snapshotTag.HighestHeight = 0
		snapshotTag.SenderPubkey = ""
		ssreceiver.snapshotTag = snapshotTag
	}
	ssreceiver.snapshotTag = snapshotTag
}

// Do NOT block PSConn goroutine when apply snapshot
func (ssreceiver *MolassesSnapshotReceiver) ApplySnapshot(s *quorumpb.Snapshot) error {
	snapshotreceiver_log.Debugf("<%s> ApplySnapshot called", ssreceiver.groupId)

	//check if all snapshots are well received
	if _, ok := ssreceiver.snapshotpackages[s.SnapshotPackageId]; ok {
		//already receive snapshot with same SnapshotPackageId
		snapshotpackage, _ := ssreceiver.snapshotpackages[s.SnapshotPackageId]
		//check if snapshot is valid
		for _, received := range snapshotpackage {
			if received.TotalCount != s.TotalCount ||
				received.HighestBlockId != s.HighestBlockId ||
				received.HighestHeight != s.HighestHeight ||
				received.TimeStamp != s.TimeStamp ||
				received.Nonce != s.Nonce {
				//drop this snapshot and clear all snapshots with the same snapshotpackageId
				snapshotreceiver_log.Warnf("<%s> Invalid snapshot, snapshotId <%s>, snapshot package id <%s>, drop all snapshots with same snapshotId", ssreceiver.groupId, s.SnapshotId, s.SnapshotPackageId)
				delete(ssreceiver.snapshotpackages, s.SnapshotPackageId)
				return nil
			}
		}
		//add new snapshot to snapshot package
		snapshotpackage[s.SnapshotId] = s
	} else {
		//create new snapshot package
		var snapshotpackage map[string]*quorumpb.Snapshot
		snapshotpackage = make(map[string]*quorumpb.Snapshot)
		//add snapshot to package
		snapshotpackage[s.SnapshotId] = s
		//add new snapshot packages
		ssreceiver.snapshotpackages[s.SnapshotPackageId] = snapshotpackage
	}

	snapshotpackage, _ := ssreceiver.snapshotpackages[s.SnapshotPackageId]

	if len(snapshotpackage) == int(s.TotalCount) {
		snapshotreceiver_log.Debugf("<%s> try apply snapshot", s.GroupId)
		if ssreceiver.snapshotTag.HighestBlockId == s.HighestBlockId &&
			ssreceiver.snapshotTag.HighestHeight == s.HighestHeight {
			snapshotreceiver_log.Debugf("<%s> snapshot already applied, only update snapshot tag", s.GroupId)
		} else {
			go ssreceiver.doApply(snapshotpackage, s)
		}
	}

	return nil
}

func (ssreceiver *MolassesSnapshotReceiver) VerifySignature(s *quorumpb.Snapshot) (bool, error) {
	snapshotreceiver_log.Debugf("<%s> VerifySignature called", ssreceiver.groupId)
	var sig []byte
	sig = s.Singature
	s.Singature = nil
	bbytes, err := proto.Marshal(s)
	if err != nil {
		return false, err
	}
	hashed := localcrypto.Hash(bbytes)

	s.Singature = sig

	ks := localcrypto.GetKeystore()
	pubkeyBytes, err := base64.RawURLEncoding.DecodeString(s.SenderPubkey)
	if err == nil {
		ethpubkey, err := ethcrypto.DecompressPubkey(pubkeyBytes)
		if err == nil {
			verify := ks.EthVerifySign(hashed, sig, ethpubkey)
			return verify, nil
		}

	}
	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(s.SenderPubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	p2pkeyverify, err := pubkey.Verify(hashed, sig)

	return p2pkeyverify, nil
}

func (ssreceiver *MolassesSnapshotReceiver) GetTag() *quorumpb.SnapShotTag {
	return ssreceiver.snapshotTag
}

func (ssreceiver *MolassesSnapshotReceiver) doApply(snapshots map[string]*quorumpb.Snapshot, s *quorumpb.Snapshot) error {
	ssreceiver.applySnapshotMu.Lock()
	defer ssreceiver.applySnapshotMu.Unlock()

	snapshotreceiver_log.Debugf("<%s> apply called", ssreceiver.groupId)
	for _, snapshot := range snapshots {
		for _, snapshotdata := range snapshot.SnapshotItems {
			if snapshotdata.Type == quorumpb.SnapShotItemType_SNAPSHOT_APP_CONFIG {
				err := nodectx.GetNodeCtx().GetChainStorage().UpdateAppConfig(snapshotdata.Data, ssreceiver.nodename)
				if err != nil {
					snapshotreceiver_log.Warningf("<%s> applySnapshot failed, type APP_CONFIG, err <%s>", ssreceiver.groupId, err.Error())
					return err
				}
			} else if snapshotdata.Type == quorumpb.SnapShotItemType_SNAPSHOT_CHAIN_CONFIG {
				err := nodectx.GetNodeCtx().GetChainStorage().UpdateChainConfig(snapshotdata.Data, ssreceiver.nodename)
				if err != nil {
					snapshotreceiver_log.Warningf("<%s> applySnapshot failed, type CHAIN_CONFIG, err <%s>", ssreceiver.groupId, err.Error())
					return err
				}
			} else if snapshotdata.Type == quorumpb.SnapShotItemType_SNAPSHOT_ANNOUNCE {
				err := nodectx.GetNodeCtx().GetChainStorage().UpdateAnnounce(snapshotdata.Data, ssreceiver.nodename)
				if err != nil {
					snapshotreceiver_log.Warningf("<%s> applySnapshot failed, type ANNOUNCE, err <%s>", ssreceiver.groupId, err.Error())
					return err
				}
			} else if snapshotdata.Type == quorumpb.SnapShotItemType_SNAPSHOT_PRODUCER {
				err := nodectx.GetNodeCtx().GetChainStorage().UpdateProducer(snapshotdata.Data, ssreceiver.nodename)
				if err != nil {
					snapshotreceiver_log.Warningf("<%s> applySnapshot failed, type PRODUCER, err <%s>", ssreceiver.groupId, err.Error())
					return err
				}
			} else if snapshotdata.Type == quorumpb.SnapShotItemType_SNAPSHOT_USER {
				err := nodectx.GetNodeCtx().GetChainStorage().UpdateUser(snapshotdata.Data, ssreceiver.nodename)
				if err != nil {
					snapshotreceiver_log.Warningf("<%s> applySnapshot failed, type USE, err <%s>", ssreceiver.groupId, err.Error())
					return err
				}
			} else {
				snapshotreceiver_log.Warningf("<%s> Unknown snapshot data type", ssreceiver.groupId)
				return errors.New("Unknown snapshot data type")
			}
		}
	}

	//update snapshotTag
	ssreceiver.snapshotTag.TimeStamp = s.TimeStamp
	ssreceiver.snapshotTag.HighestHeight = s.HighestHeight
	ssreceiver.snapshotTag.HighestBlockId = s.HighestBlockId
	ssreceiver.snapshotTag.Nonce = s.Nonce
	ssreceiver.snapshotTag.SnapshotPackageId = s.SnapshotPackageId
	ssreceiver.snapshotTag.SenderPubkey = s.SenderPubkey

	err := nodectx.GetNodeCtx().GetChainStorage().UpdateSnapshotTag(ssreceiver.groupId, ssreceiver.snapshotTag, ssreceiver.nodename)
	if err != nil {
		snapshotreceiver_log.Warningf("<%s> UpdateSnapshotTag failed, err <%s>", ssreceiver.groupId, err.Error())
		return err
	}

	//remove snapshot package
	delete(ssreceiver.snapshotpackages, s.SnapshotPackageId)

	return nil
}

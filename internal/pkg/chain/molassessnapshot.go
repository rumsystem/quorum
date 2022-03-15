package chain

import (
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

var snapshot_log = logging.Logger("snapshot")

const DEFAULT_SNAPSHOT_INTERVAL time.Duration = 60 //60s
const PACKAGE_TOTAL_SIZE int = 900 * 1024          //Maximum package payload is 900K

type MolassesSnapshot struct {
	grpItem       *quorumpb.GroupItem
	cIface        ChainMolassesIface
	nodename      string
	SnapshotTimer *time.Timer
	groupId       string
}

func (snapshot *MolassesSnapshot) Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface) {
	snapshot_log.Debug("Init called")
	snapshot.grpItem = item
	snapshot.cIface = iface
	snapshot.nodename = nodename
	snapshot.groupId = item.GroupId

	snapshot_log.Infof("<%s> producer created", snapshot.groupId)
}

func (snapshot *MolassesSnapshot) SetInterval(sec int) {

}

func (snapshot *MolassesSnapshot) Start() error {
	return nil
}

func (snapshot *MolassesSnapshot) Stop() error {
	return nil
}

func (snapshot *MolassesSnapshot) GetSnapshot() ([]*quorumpb.SnapShotItem, error) {
	snapshot_log.Debugf("<%s> GetSnapshot called", snapshot.groupId)
	var appconfig [][]byte
	appconfig, err := nodectx.GetDbMgr().GetAllAppConfigInBytes(snapshot.groupId, snapshot.nodename)
	if err != nil


	return nil, nil
}

func (snapshot *MolassesSnapshot) sendSnapshot() error {
	snapshot_log.Debugf("<%s> sendSnapshot called", snapshot.groupId)
	return nil
}

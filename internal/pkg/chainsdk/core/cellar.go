package chain

import (
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var cellar_log = logging.Logger("cellar")

type Cellar struct {
	CellarId    string
	CellarGroup *Group
}

func (clr *Cellar) JoinCellarBySeed(userPubkey string, seed *quorumpb.CellarSeed) error {
	cellar_log.Debugf("JoinCellarBySeed called")
	return nil
}

func (clr *Cellar) LoadCellar(item *quorumpb.CellarItem) error {
	cellar_log.Debugf("LoadCellar called")
	return nil
}

func (clr *Cellar) LoadCellarById(cellarId string) error {
	cellar_log.Debugf("LoadCellarById called")
	return nil
}

func (clr *Cellar) Teardown() error {
	cellar_log.Debugf("Teardown called")
	return nil
}

func (clr *Cellar) LeaveCellar() error {
	cellar_log.Debugf("LeaveCella called")
	return nil
}

func (clr *Cellar) RmCellarData() error {
	cellar_log.Debugf("RmCellarData called")
	return nil
}

func (clr *Cellar) GetCellarGroup() *Group {
	cellar_log.Debugf("GetCellarInfo called")
	return clr.CellarGroup
}

func (clr *Cellar) AddGroupBySeed(seed *quorumpb.GroupSeed) error {
	cellar_log.Debugf("UpdCellarGroup called")
	return nil
}

func (clr *Cellar) RmGroupById(groupId string) error {
	cellar_log.Debugf("RmGroupById called")
	return nil
}

package chain

import (
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

var cellarMgr_log = logging.Logger("cellarmgr")

type CellarMgr struct {
	Cellars map[string]*Cellar
}

var cellarMgr *CellarMgr

func GetCellarMgr() *CellarMgr {
	return cellarMgr
}

func InitCellarMgr() error {
	cellarMgr_log.Debug("InitCellarMgr called")
	cellarMgr = &CellarMgr{}
	cellarMgr.Cellars = make(map[string]*Cellar)
	return nil
}

func (cellarMgr *CellarMgr) LoadAllCellars() error {
	cellarMgr_log.Debug("LoadAllCellars called")
	cellarItems, err := nodectx.GetNodeCtx().GetChainStorage().GetAllCellars()
	if err != nil {
		return err
	}

	for _, cellarItem := range cellarItems {
		cellarMgr_log.Debugf("load cellar: %s", cellarItem.CellarId)
		cellar := &Cellar{}
		cellar.LoadCellar(cellarItem)
		cellarMgr.Cellars[cellar.CellarId] = cellar
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (cellarMgr *CellarMgr) TeardownAllCellars() error {
	cellarMgr_log.Debug("TeardownAllCellars called")
	for _, cellar := range cellarMgr.Cellars {
		cellar.Teardown()
	}
	return nil
}

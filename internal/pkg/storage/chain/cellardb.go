package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) AddCellar(cellarItem *quorumpb.CellarItem) error {
	key := s.GetCellarKey(cellarItem.CellarId)
	value, err := proto.Marshal(cellarItem)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) UpdCellar(cellarItem *quorumpb.CellarItem) error {
	return cs.AddCellar(cellarItem)
}

func (cs *Storage) RmCellar(cellarId string) error {
	key := s.GetCellarKey(cellarId)
	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return nil
	}
	return cs.dbmgr.Db.Delete([]byte(key))
}

func (cs *Storage) GetCellarInfo(cellarId string) (*quorumpb.CellarItem, error) {
	key := s.GetCellarKey(cellarId)
	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	cellarItem := &quorumpb.CellarItem{}
	err = proto.Unmarshal(value, cellarItem)
	if err != nil {
		return nil, err
	}
	return cellarItem, nil
}

func (cs *Storage) RmCellarData(cellarId string) error {
	//TBD
	//remove all groupitems
	//remove cellar
	return nil
}

func (cs *Storage) IsCellarExist(cellarId string) (bool, error) {
	key := s.GetCellarKey(cellarId)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

func (cs *Storage) GetAllCellars() ([]*quorumpb.CellarItem, error) {
	key := s.GetCellarPrefix()
	cellars := []*quorumpb.CellarItem{}
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		cellarItem := &quorumpb.CellarItem{}
		err = proto.Unmarshal(v, cellarItem)
		if err != nil {
			return err
		}
		cellars = append(cellars, cellarItem)
		return nil
	})

	return cellars, err
}

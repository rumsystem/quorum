package chainstorage

import (
	"errors"

	"github.com/golang/protobuf/proto"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
)

func (cs *Storage) AddTrxHBB(trx *quorumpb.Trx, queueId string) error {
	key := s.CNS_BUFD_TRX + "_" + queueId + "_" + trx.TrxId

	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}

	if exist {
		return errors.New("Trx exist")
	}

	if value, err := proto.Marshal(trx); err != nil {
		return err
	} else {
		err = cs.dbmgr.Db.Set([]byte(key), value)
		return err
	}
}

func (cs *Storage) GetAllTrxHBB(queueId string) ([]*quorumpb.Trx, error) {
	var trxs []*quorumpb.Trx
	key := s.CNS_BUFD_TRX + "_" + queueId

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		trx := &quorumpb.Trx{}
		if err := proto.Unmarshal(v, trx); err != nil {
			return err
		}

		trxs = append(trxs, trx)
		return nil
	})
	return trxs, err
}

func (cs *Storage) GeBufferedTrxLenHBB(queueId string) (int, error) {
	trxs, err := cs.GetAllTrxHBB(queueId)
	return len(trxs), err
}

func (cs *Storage) RemoveTrxHBB(trxId, queueId string) error {
	key := s.CNS_BUFD_TRX + "_" + queueId + "_" + trxId
	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}

	if !exist {
		return errors.New("Trx not exist")
	}

	_, err = cs.dbmgr.Db.PrefixDelete([]byte(key))
	return err
}

func (cs *Storage) RemoveAllTrxHBB(queueId string) error {
	key_prefix := s.CNS_BUFD_TRX + "_" + queueId + "_"
	_, err := cs.dbmgr.Db.PrefixDelete([]byte(key_prefix))
	return err
}

func (cs *Storage) GetTrxByIdHBB(trxId string, queueId string) (*quorumpb.Trx, error) {
	key := s.CNS_BUFD_TRX + "_" + queueId + "_" + trxId

	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, errors.New("Trx is not exist")
	}

	trxInBytes, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	trx := &quorumpb.Trx{}
	if err := proto.Unmarshal(trxInBytes, trx); err != nil {
		return nil, err
	}

	return trx, nil
}

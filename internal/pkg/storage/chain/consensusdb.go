package chainstorage

import (
	"encoding/binary"
	"errors"

	"github.com/golang/protobuf/proto"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

func (cs *Storage) AddTrxHBB(trx *quorumpb.Trx, queueId string) error {
	key := s.GetTrxHBBKey(queueId, trx.TrxId)
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
	key := s.GetTrxHBBPrefix(queueId)
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
	if err != nil {
		return -1, err
	}
	return len(trxs), nil
}

func (cs *Storage) RemoveTrxHBB(trxId, queueId string) error {
	key := s.GetTrxHBBKey(queueId, trxId)
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
	key_prefix := s.GetTrxHBBPrefix(queueId)
	_, err := cs.dbmgr.Db.PrefixDelete([]byte(key_prefix))
	return err
}

func (cs *Storage) GetTrxByIdHBB(trxId string, queueId string) (*quorumpb.Trx, error) {
	key := s.GetTrxHBBKey(queueId, trxId)

	exist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, errors.New("trx is not exist")
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
<<<<<<< HEAD

func (cs *Storage) AddConsensusProposeNonce(queueId string, nonce uint64) error {
	key := s.GetConsensusNonceKey(queueId)
	e := make([]byte, 8)
	binary.LittleEndian.PutUint64(e, nonce)
	cs.dbmgr.Db.Set([]byte(key), e)
	return cs.dbmgr.Db.Set([]byte(key), e)
}

func (cs *Storage) GetConsensusProposeNonce(queueId string) (uint64, error) {
	key := s.GetConsensusNonceKey(queueId)
	nonceInBytes, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return 0, err
	}

	nonce := binary.LittleEndian.Uint64(nonceInBytes)
	return nonce, nil
}
=======
>>>>>>> consensus_2_main

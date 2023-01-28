package chainstorage

import (
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
	return len(trxs), err
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

func (cs *Storage) IsPSyncSessionExist(groupId, sessionId string) (bool, error) {
	key := s.GetPSyncKey(groupId, sessionId)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

func (cs *Storage) UpdPSyncResp(groupId, sessionId string, resp *quorumpb.PSyncResp) error {
	//remove all current group PSync Session
	key_prefix := s.GetPSyncPrefix(groupId)
	_, err := cs.dbmgr.Db.PrefixDelete([]byte(key_prefix))
	if err != nil {
		return err
	}

	//update group psync session
	key := s.GetPSyncKey(groupId, sessionId)
	if value, err := proto.Marshal(resp); err != nil {
		return err
	} else {
		err = cs.dbmgr.Db.Set([]byte(key), value)
		return err
	}
}

func (cs *Storage) GetCurrentPSyncSession(groupId string) ([]*quorumpb.PSyncResp, error) {

	/*
		resps := []*quorumpb.PSyncResp{}
		key := s.GetPSyncPrefix(groupId)
		err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}

			resp := &quorumpb.ConsensusResp{}
			if err := proto.Unmarshal(v, resp); err != nil {
				return err
			}

			//should be only 1 (or no) resp item, otherwise something goes wrong
			resps = append(resps, resp)
			return nil
		})

		return resps, err
	*/

	return nil, nil
}

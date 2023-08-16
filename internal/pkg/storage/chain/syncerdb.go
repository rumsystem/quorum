package chainstorage

import (
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateGroupSyncer(trxId string, data []byte, prefix ...string) (err error) {
	item := &quorumpb.UpdGroupSyncerItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	syncer := item.Syncer

	syncerKey := s.GetSyncerKey(syncer.GroupId, syncer.SyncerPubkey, prefix...)
	exist, _ := cs.dbmgr.Db.IsExist([]byte(syncerKey))

	if item.Action == quorumpb.ActionType_ADD {
		if exist {
			return errors.New("syncer already exist")
		}

		//save it to db
		syncerBytes, err := proto.Marshal(syncer)
		if err != nil {
			return err
		}
		return cs.dbmgr.Db.Set([]byte(syncerKey), syncerBytes)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		if !exist {
			return errors.New("syncer not exist")
		}

		//delete it from db
		return cs.dbmgr.Db.Delete([]byte(syncerKey))
	} else {
		chaindb_log.Infof("unknow msgType")
		return errors.New("unknow msgType")
	}
}

func (cs *Storage) GetSyncers(groupId string, prefix ...string) ([]*quorumpb.Syncer, error) {
	var sList []*quorumpb.Syncer

	key := s.GetSyncerPrefix(groupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.Syncer{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		sList = append(sList, &item)
		return nil
	})

	return sList, err
}

func (cs *Storage) GetSyncer(groupId string, pubkey string, prefix ...string) (*quorumpb.Syncer, error) {
	key := s.GetSyncerKey(groupId, pubkey, prefix...)
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var syncer quorumpb.Syncer
	err = proto.Unmarshal(value, &syncer)
	if err != nil {
		return nil, err
	}

	return &syncer, err
}

func (cs *Storage) IsSyncer(groupId, userSignPubkey string, prefix ...string) (bool, error) {
	key := s.GetSyncerKey(groupId, userSignPubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

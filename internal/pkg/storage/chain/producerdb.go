package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) AddProducer(item *quorumpb.Producer, prefix ...string) error {
	key := s.GetProducerKey(item.GroupId, item.ProducerPubkey, prefix...)
	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), pbyte)
}

func (cs *Storage) RemoveAllProducers(groupId string, prefix ...string) error {
	key := s.GetProducerPrefix(groupId, prefix...)
	_, err := cs.dbmgr.Db.PrefixDelete([]byte(key))
	return err
}

func (cs *Storage) GetProducers(groupId string, prefix ...string) ([]*quorumpb.Producer, error) {
	var pList []*quorumpb.Producer
	key := s.GetProducerPrefix(groupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		item := &quorumpb.Producer{}
		perr := proto.Unmarshal(v, item)
		if perr != nil {
			return perr
		}
		pList = append(pList, item)
		return nil
	})
	return pList, err
}

func (cs *Storage) GetProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.Producer, error) {
	key := s.GetProducerKey(groupId, pubkey, prefix...)

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var producer quorumpb.Producer
	err = proto.Unmarshal(value, &producer)
	if err != nil {
		return nil, err
	}

	return &producer, err
}

func (cs *Storage) IsProducerExist(groupId, pubkey string, prefix ...string) (bool, error) {
	key := s.GetProducerKey(groupId, pubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

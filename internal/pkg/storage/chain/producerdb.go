package chainstorage

import (
	"encoding/binary"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) AddProducer(item *quorumpb.ProducerItem, prefix ...string) error {
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.ProducerPubkey)
	if pk == "" {
		pk = item.ProducerPubkey
	}

	key := s.GetProducerKey(item.GroupId, pk, prefix...)
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

func (cs *Storage) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var pList []*quorumpb.ProducerItem
	key := s.GetProducerPrefix(groupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		item := &quorumpb.ProducerItem{}
		perr := proto.Unmarshal(v, item)
		if perr != nil {
			return perr
		}
		pList = append(pList, item)
		return nil
	})

	return pList, err
}

func (cs *Storage) GetProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.ProducerItem, error) {
	key := s.GetProducerKey(groupId, pubkey, prefix...)

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var producer quorumpb.ProducerItem
	err = proto.Unmarshal(value, &producer)
	if err != nil {
		return nil, err
	}

	return &producer, err
}

func (cs *Storage) IsProducer(groupId, pubkey string, prefix ...string) (bool, error) {
	key := s.GetProducerKey(groupId, pubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

func (cs *Storage) GetProducerConsensusConfInterval(groupId string, prefix ...string) (uint64, error) {
	key := s.GetProducerConsensusConfInterval(groupId, prefix...)
	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return 0, err
	}

	interval := uint64(binary.LittleEndian.Uint64(value))
	return interval, nil
}

func (cs *Storage) SetProducerConsensusConfInterval(groupId string, proposeTrxInterval uint64, prefix ...string) error {
	key := s.GetProducerConsensusConfInterval(groupId, prefix...)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, proposeTrxInterval)
	return cs.dbmgr.Db.Set([]byte(key), b)
}

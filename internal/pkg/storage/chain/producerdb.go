package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateProducer(trxId string, data []byte, prefix ...string) error {
	return nil
}

func (cs *Storage) AddProducer(item *quorumpb.ProducerItem, prefix ...string) error {
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.ProducerPubkey)
	if pk == "" {
		pk = item.ProducerPubkey
	}

	key := s.GetProducerKey(item.GroupId, pk, prefix...)
	chaindb_log.Infof("Add Producer with key %s", key)

	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), pbyte)
}

func (cs *Storage) GetAnnouncedProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	key := s.GetAnnounceAsProducerKey(groupId, pubkey, prefix...)

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var ann quorumpb.AnnounceItem
	err = proto.Unmarshal(value, &ann)
	if err != nil {
		return nil, err
	}

	return &ann, err
}

func (cs *Storage) GetAnnouncedProducers(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem
	key := s.GetAnnounceAsProducerPrefix(groupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.AnnounceItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		aList = append(aList, &item)
		return nil
	})

	return aList, err
}

func (cs *Storage) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var pList []*quorumpb.ProducerItem
	key := s.GetProducerPrefix(groupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.ProducerItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		pList = append(pList, &item)
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

func (cs *Storage) IsProducerAnnounced(groupId, pubkey string, prefix ...string) (bool, error) {
	key := s.GetAnnounceAsProducerKey(groupId, pubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

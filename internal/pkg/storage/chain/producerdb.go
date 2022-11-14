package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateProducerTrx(trx *quorumpb.Trx, prefix ...string) error {
	err := cs.UpdateProducer(trx.GroupId, trx.Data, prefix...)
	if err != nil {
		return err
	}

	//save trxId of latest producer update trx
	groupInfo, err := cs.GetGroupInfo(trx.GroupId)
	if err != nil {
		return err
	}

	key := s.GetProducerTrxIDKey(groupInfo.GroupId, prefix...)
	return cs.dbmgr.Db.Set([]byte(key), []byte(trx.TrxId))
}

func (cs *Storage) GetUpdProducerListTrx(groupId string, prefix ...string) (*quorumpb.Trx, error) {
	key := s.GetProducerTrxIDKey(groupId, prefix...)
	btrx_id, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	trxId := string(btrx_id)

	trx, _, err := cs.GetTrx(groupId, trxId, def.Chain, prefix...)
	if err != nil {
		return nil, err
	}

	return trx, nil
}

func (cs *Storage) UpdateProducer(groupId string, data []byte, prefix ...string) error {
	item := &quorumpb.BFTProducerBundleItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	groupInfo, err := cs.GetGroupInfo(groupId)
	if err != nil {
		return err
	}

	//Get all current producers (except owner)
	var cplist []string
	key := s.GetProducerPrefix(groupId, prefix...)
	err = cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := &quorumpb.ProducerItem{}
		perr := proto.Unmarshal(v, item)
		if perr != nil {
			return perr
		}

		if item.ProducerPubkey != groupInfo.OwnerPubKey {
			pkey := key + "_" + item.ProducerPubkey
			cplist = append(cplist, pkey)
		}

		return nil
	})

	if err != nil {
		return err
	}

	//remove all producers (except owner)
	for _, pkey := range cplist {
		err := cs.dbmgr.Db.Delete([]byte(pkey))
		if err != nil {
			return err
		}
	}

	//update with new producers list
	for _, producerItem := range item.Producers {
		pk, _ := localcrypto.Libp2pPubkeyToEthBase64(producerItem.ProducerPubkey)
		if pk == "" {
			pk = producerItem.ProducerPubkey
		}

		pdata, err := proto.Marshal(producerItem)
		if err != nil {
			return err
		}

		key := s.GetProducerKey(producerItem.GroupId, pk, prefix...)
		err = cs.dbmgr.Db.Set([]byte(key), pdata)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cs *Storage) GetAllProducerInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	key := s.GetProducerPrefix(groupId, Prefix...)
	var producerByteList [][]byte

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		producerByteList = append(producerByteList, v)
		return nil
	})

	return producerByteList, err
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

func (cs *Storage) IsProducerAnnounced(groupId, pubkey string, prefix ...string) (bool, error) {
	key := s.GetAnnounceAsProducerKey(groupId, pubkey, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

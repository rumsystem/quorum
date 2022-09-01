package chainstorage

import (
	"errors"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateProducerTrx(trx *quorumpb.Trx, prefix ...string) (err error) {
	return cs.UpdateProducer(trx.Data, prefix...)
}

func (cs *Storage) UpdateProducer(data []byte, prefix ...string) (err error) {
	nodeprefix := utils.GetPrefix(prefix...)
	item := &quorumpb.BFTProducerBundleItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	// TBD need modify
	for _, producerItem := range item.Producers {
		pk, _ := localcrypto.Libp2pPubkeyToEthBase64(producerItem.ProducerPubkey)
		if pk == "" {
			pk = producerItem.ProducerPubkey
		}

		pdata, err := proto.Marshal(producerItem)
		if err != nil {
			return err
		}

		key := nodeprefix + s.PRD_PREFIX + "_" + producerItem.GroupId + "_" + pk
		if producerItem.Action == quorumpb.ActionType_ADD {
			err := cs.dbmgr.Db.Set([]byte(key), pdata)
			if err != nil {
				return err
			}
		} else if producerItem.Action == quorumpb.ActionType_REMOVE {
			//check if group exist
			chaindb_log.Infof("Remove producer")
			exist, err := cs.dbmgr.Db.IsExist([]byte(key))
			if !exist {
				if err != nil {
					return err
				}
				return errors.New("Producer Not Found")
			}
			err = cs.dbmgr.Db.Delete([]byte(key))
			if err != nil {
				return nil
			}
		} else {
			return errors.New("unknow msgType")
		}
	}

	return nil
}

func (cs *Storage) GetAllProducerInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + s.PRD_PREFIX + "_" + groupId + "_"
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

	nodeprefix := utils.GetPrefix(prefix...)

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.ProducerPubkey)
	if pk == "" {
		pk = item.ProducerPubkey
	}

	key := nodeprefix + s.PRD_PREFIX + "_" + item.GroupId + "_" + pk
	chaindb_log.Infof("Add Producer with key %s", key)

	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), pbyte)
}

// commented by cuicat
// all producer change to witenesses, no need to count block produced
/*
func (cs *Storage) AddProducedBlockCount(groupId, pubkey string, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(pubkey)

	var err error
	libp2ppk := ""
	if pk == pubkey {
		libp2ppk, err = localcrypto.EthBase64ToLibp2pPubkey(pubkey)
	} else if pk == "" {
		pk = pubkey
	}

	key := nodeprefix + s.PRD_PREFIX + "_" + groupId + "_" + pk
	var pProducer *quorumpb.ProducerItem
	pProducer = &quorumpb.ProducerItem{}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		if pubkey != "" {
			//patch for old keyformat
			key = nodeprefix + s.PRD_PREFIX + "_" + groupId + "_" + libp2ppk
			value, err = cs.dbmgr.Db.Get([]byte(key))
			if err != nil {
				key = s.PRD_PREFIX + "_" + groupId + "_" + libp2ppk
				value, err = cs.dbmgr.Db.Get([]byte(key))
				if err != nil {
					return err
				}

			}
			//update to the new keyformat
			key = nodeprefix + s.PRD_PREFIX + "_" + groupId + "_" + pk
		} else {
			return err
		}

	}

	err = proto.Unmarshal(value, pProducer)
	if err != nil {
		return err
	}

	pProducer.BlockProduced += 1

	value, err = proto.Marshal(pProducer)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), value)
}
*/
func (cs *Storage) GetAnnouncedProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(pubkey)
	if pk == "" {
		pk = pubkey
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + pk

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
	nodeprefix := utils.GetPrefix(prefix...)
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(pubkey)
	if pk == "" {
		pk = pubkey
	}
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + pk
	return cs.dbmgr.Db.IsExist([]byte(key))
}

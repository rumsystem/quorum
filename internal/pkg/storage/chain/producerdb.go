package chainstorage

import (
	"errors"
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
	item := &quorumpb.ProducerItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}

	key := nodeprefix + s.PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey

	chaindb_log.Infof("upd producer with key %s", key)

	if item.Action == quorumpb.ActionType_ADD {
		chaindb_log.Infof("Add producer")
		return cs.dbmgr.Db.Set([]byte(key), data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if group exist
		chaindb_log.Infof("Remove producer")
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Producer Not Found")
		}

		return cs.dbmgr.Db.Delete([]byte(key))
	} else {
		chaindb_log.Infof("Remove producer")
		return errors.New("unknow msgType")
	}
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
	key := nodeprefix + s.PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey
	chaindb_log.Infof("Add Producer with key %s", key)

	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	return cs.dbmgr.Db.Set([]byte(key), pbyte)
}

func (cs *Storage) AddProducedBlockCount(groupId, producerPubkey string, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.PRD_PREFIX + "_" + groupId + "_" + producerPubkey
	var pProducer *quorumpb.ProducerItem
	pProducer = &quorumpb.ProducerItem{}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return err
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

func (cs *Storage) GetAnnouncedProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + pubkey

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

func (cs *Storage) IsProducerAnnounced(groupId, producerSignPubkey string, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + producerSignPubkey
	return cs.dbmgr.Db.IsExist([]byte(key))
}

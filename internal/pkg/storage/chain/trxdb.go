package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var logger = logging.Logger("chainstorage")

// save trx
func (cs *Storage) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	key := s.GetTrxKey(trx.GroupId, trx.TrxId, prefix...)
	value, err := proto.Marshal(trx)
	if err != nil {
		return err
	}
	err = cs.dbmgr.Db.Set([]byte(key), value)
	return err
}

func (cs *Storage) UpdTrx(trx *quorumpb.Trx, prefix ...string) error {
	return cs.AddTrx(trx, prefix...)
}

// Get Trx
func (cs *Storage) GetTrx(groupId string, trxId string, storagetype def.TrxStorageType, prefix ...string) (t *quorumpb.Trx, err error) {
	trx := &quorumpb.Trx{}
	var key string

	if storagetype == def.Chain {
		key = s.GetTrxKey(groupId, trxId, prefix...)
		value, err := cs.dbmgr.Db.Get([]byte(key))
		err = proto.Unmarshal(value, trx)
		if err != nil {
			return nil, err
		}
		trx.StorageType = quorumpb.TrxStroageType_CHAIN
	} else if storagetype == def.Cache {
		key = s.GetCachedBlockPrefix(groupId, prefix...)
		err = cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				logger.Errorf("cs.dbmgr.Db.PrefixForeach failed: %s", err)
				return err
			}
			block := quorumpb.Block{}
			perr := proto.Unmarshal(v, &block)
			if perr != nil {
				logger.Errorf("proto.Unmarshal chunk failed: %s", err)
				return perr
			}
			if block.Trxs != nil {
				for _, trxInBlock := range block.Trxs {
					if trxInBlock.TrxId == trxId {
						//clone trx
						cloneTrxBytes, _ := proto.Marshal(trxInBlock)
						proto.Unmarshal(cloneTrxBytes, trx)
						trx.StorageType = quorumpb.TrxStroageType_CACHE
						return nil
					}
				}
			}

			return nil
		})
	}

	//convert pubkey to base64
	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(trx.SenderPubkey)
	trx.SenderPubkey = pk

	return trx, err
}

func (cs *Storage) IsTrxExist(groupId string, trxId string, prefix ...string) (bool, error) {
	key := s.GetTrxKey(groupId, trxId, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

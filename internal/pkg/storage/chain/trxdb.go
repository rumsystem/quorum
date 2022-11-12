package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

// save trx
func (cs *Storage) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	key := s.GetTrxKey(trx.TrxId, trx.Nonce, prefix...)
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
func (cs *Storage) GetTrx(trxId string, storagetype def.TrxStorageType, prefix ...string) (t *quorumpb.Trx, n []int64, err error) {
	var trx quorumpb.Trx
	var nonces []int64

	var key string
	if storagetype == def.Chain {
		key = s.GetTrxPrefix(trxId, prefix...)
		err = cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			perr := proto.Unmarshal(v, &trx)
			if perr != nil {
				return perr
			}
			nonces = append(nonces, trx.Nonce)
			return nil
		})
		trx.StorageType = quorumpb.TrxStroageType_CHAIN
	} else if storagetype == def.Cache {
		key = s.GetCachedBlockPrefix(prefix...)
		err = cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			block := quorumpb.Block{}
			perr := proto.Unmarshal(v, &block)
			if perr != nil {
				return perr
			}
			if block.Trxs != nil {
				for _, blocktrx := range block.Trxs {
					if blocktrx.TrxId == trxId {
						nonces = append(nonces, blocktrx.Nonce)

						clonedtrxbuff, _ := proto.Marshal(blocktrx)
						perr = proto.Unmarshal(clonedtrxbuff, &trx)
						if perr != nil {
							return perr
						}
						trx.StorageType = quorumpb.TrxStroageType_CACHE
						return nil
					}
				}
			}

			return nil
		})

	}

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(trx.SenderPubkey)
	trx.SenderPubkey = pk
	return &trx, nonces, err
}

func (cs *Storage) IsTrxExist(trxId string, nonce int64, prefix ...string) (bool, error) {
	key := s.GetTrxKey(trxId, nonce, prefix...)
	return cs.dbmgr.Db.IsExist([]byte(key))
}

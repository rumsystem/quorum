package appdata

import (
	"fmt"
	badger "github.com/dgraph-io/badger/v3"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
)

var appdatalog = logging.Logger("appdata")

const GRP_PREFIX string = "grp_"
const CNT_PREFIX string = "cnt_"
const SDR_PREFIX string = "sdr_"
const SEQ_PREFIX string = "seq_"
const TRX_PREFIX string = "trx_"

type AppDb struct {
	Db       *badger.DB
	seq      map[string]*badger.Sequence
	DataPath string
}

func InitDb(datapath string, dbopts *chain.DbOption) *AppDb {
	newdb, err := badger.Open(badger.DefaultOptions(datapath + "_appdb").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression))
	if err != nil {
		appdatalog.Fatal(err.Error())
	}
	seq := make(map[string]*badger.Sequence)
	appdatalog.Infof("Appdb initialized")
	return &AppDb{Db: newdb, DataPath: datapath, seq: seq}
}

func (appdb *AppDb) GetSeqId(seqkey string) (uint64, error) {
	var err error
	if appdb.seq[seqkey] == nil {
		appdb.seq[seqkey], err = appdb.Db.GetSequence([]byte(seqkey), 100)
		if err != nil {
			return 0, err
		}
	}

	return appdb.seq[seqkey].Next()
}

func (appdb *AppDb) Rebuild(vertag string, chainDb *badger.DB) error {

	return nil
}

func (appdb *AppDb) GetData(prefix string, start uint64, num uint64) error {
	//key := GRP_PREFIX + CNT_PREFIX + groupId + "_"
	//"cnt_grp_9adced87-3381-404b-8282-489278662c16"
	err := appdb.Db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			k := item.Key()
			fmt.Printf("key=%s\n", k)
			//err := item.Value(func(v []byte) error {
			//	contentitem := &quorumpb.PostItem{}
			//	ctnerr := proto.Unmarshal(v, contentitem)
			//	if ctnerr == nil {
			//		ctnList = append(ctnList, contentitem)
			//	}
			//	return ctnerr
			//})

			//if err != nil {
			//	return err
			//}
		}

		return nil
	})
	return err
}

func (appdb *AppDb) AddMetaByTrx(trx *quorumpb.Trx) error {
	var err error

	seqkey := SEQ_PREFIX + CNT_PREFIX + GRP_PREFIX + trx.GroupId
	seqid, err := appdb.GetSeqId(seqkey)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s%s_%d_%s", CNT_PREFIX, GRP_PREFIX, trx.GroupId, seqid, trx.TrxId)
	err = appdb.Db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), nil)
		err := txn.SetEntry(e)
		return err
	})

	seqkey = SEQ_PREFIX + CNT_PREFIX + GRP_PREFIX + trx.GroupId + SDR_PREFIX
	seqid, err = appdb.GetSeqId(seqkey)
	if err != nil {
		return err
	}
	fmt.Println("insert key:", key)

	key = fmt.Sprintf("%s%s%s_%s_%d_%s", SDR_PREFIX, GRP_PREFIX, trx.GroupId, trx.Sender, seqid, trx.TrxId)
	err = appdb.Db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), nil)
		err := txn.SetEntry(e)
		return err
	})
	if err != nil {
		return err
	}
	fmt.Println("insert key:", key)

	//trx.TrxId
	//trx.Sender
	//Index 1: order by insert sequence
	//
	return err
}

func (appdb *AppDb) Release() error {
	for seqkey := range appdb.seq {
		appdb.seq[seqkey].Release()
	}
	return nil
}

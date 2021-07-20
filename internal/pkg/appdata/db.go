package appdata

import (
	"encoding/binary"
	"fmt"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/google/orderedcode"
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
const term = "\x00\x01"

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

func (appdb *AppDb) GetMaxBlockNum(groupid string) (uint64, error) {
	key := fmt.Sprintf("max_%s%s", GRP_PREFIX, groupid)
	var max uint64
	err := appdb.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		b, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		max = binary.LittleEndian.Uint64(b)
		return err
	})
	return max, err
}

func (appdb *AppDb) GetGroupContent(groupid string, start uint64, num int) ([]string, error) {

	prefix := fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, groupid)

	startidx := uint64(0)
	trxids := []string{}
	err := appdb.Db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 20
		opts.PrefetchValues = false
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()
		p := []byte(prefix)
		for it.Seek(append(p, 0xff)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			k := item.Key()
			if startidx >= start { //college
				trxids = append(trxids, string(k[len(k)-37-1:len(k)-1-1]))
			}
			if len(trxids) == num {
				break
			}
			startidx++
		}
		return nil
	})
	return trxids, err
}

func getKey(prefix string, seqid uint64, tailing string) ([]byte, error) {
	return orderedcode.Append(nil, prefix, "-", orderedcode.Infinity, uint64(seqid), "-", tailing)
}

func (appdb *AppDb) AddMetaByTrx(blocknum uint64, groupid string, trx *quorumpb.Trx) error {
	var err error

	seqkey := SEQ_PREFIX + CNT_PREFIX + GRP_PREFIX + trx.GroupId
	seqid, err := appdb.GetSeqId(seqkey)
	if err != nil {
		return err
	}

	key, err := getKey(fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, trx.GroupId), seqid, trx.TrxId)
	fmt.Println(key)
	fmt.Println(string(key))
	if err != nil {
		return err
	}

	seqkey1 := SEQ_PREFIX + CNT_PREFIX + GRP_PREFIX + trx.GroupId + SDR_PREFIX
	seqid1, err := appdb.GetSeqId(seqkey1)
	if err != nil {
		return err
	}

	key1, err := getKey(fmt.Sprintf("%s%s-%s", SDR_PREFIX, GRP_PREFIX, trx.GroupId), seqid1, fmt.Sprintf("%s:%s", trx.Sender, trx.TrxId))
	if err != nil {
		return err
	}

	txn := appdb.Db.NewTransaction(true)
	defer txn.Discard()

	e := badger.NewEntry(key, nil)
	err = txn.SetEntry(e)
	if err != nil {
		return err
	}

	e = badger.NewEntry(key1, nil)
	err = txn.SetEntry(e)
	if err != nil {
		return err
	}

	maxnum := make([]byte, 8)
	binary.LittleEndian.PutUint64(maxnum, uint64(blocknum))
	e = badger.NewEntry([]byte(fmt.Sprintf("max_%s%s", GRP_PREFIX, groupid)), maxnum)
	err = txn.SetEntry(e)
	if err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}
	return err
}

func (appdb *AppDb) Release() error {
	for seqkey := range appdb.seq {
		appdb.seq[seqkey].Release()
	}
	return nil
}

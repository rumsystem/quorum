package appdata

import (
	"bytes"
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

func (appdb *AppDb) GetGroupContentBySenders(groupid string, senders []string, starttrx string, num int, reverse bool) ([]string, error) {
	prefix := fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, groupid)
	sendermap := make(map[string]bool)
	for _, s := range senders {
		sendermap[s] = true
	}
	startidx := uint64(0)
	trxids := []string{}
	err := appdb.Db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 20
		opts.PrefetchValues = false
		opts.Reverse = reverse
		it := txn.NewIterator(opts)
		defer it.Close()
		p := []byte(prefix)
		if reverse == true {
			p = append(p, 0xff)
		}
		runcollector := false
		if starttrx == "" {
			runcollector = true //no trxid, start collecting from the first item
		}
		for it.Seek(p); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			k := item.Key()
			dataidx := bytes.LastIndexByte(k, byte('_'))
			trxid := string(k[len(k)-37-1 : len(k)-1-1])
			if runcollector == true {
				sender := string(k[dataidx+1+2 : len(k)-37-2]) //+2/-2 for remove the term, len(term)=2
				if len(senders) == 0 || sendermap[sender] == true {
					trxids = append(trxids, trxid)
				}
			}
			if trxid == starttrx { //start collecting after this item
				runcollector = true
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
	return orderedcode.Append(nil, prefix, "-", orderedcode.Infinity, uint64(seqid), "_", tailing)
}

func (appdb *AppDb) AddMetaByTrx(blocknum uint64, groupid string, trx *quorumpb.Trx) error {
	var err error

	seqkey := SEQ_PREFIX + CNT_PREFIX + GRP_PREFIX + trx.GroupId
	seqid, err := appdb.GetSeqId(seqkey)
	if err != nil {
		return err
	}

	key, err := getKey(fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, trx.GroupId), seqid, fmt.Sprintf("%s:%s", trx.Sender, trx.TrxId))
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

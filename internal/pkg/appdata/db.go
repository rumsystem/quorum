package appdata

import (
	"bytes"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/google/orderedcode"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
)

var appdatalog = logging.Logger("appdata")

const GRP_PREFIX string = "grp_"
const CNT_PREFIX string = "cnt_"
const SDR_PREFIX string = "sdr_"
const SEQ_PREFIX string = "seq_"
const TRX_PREFIX string = "trx_"
const STATUS_PREFIX string = "stu_"
const term = "\x00\x01"

type AppDb struct {
	Db       *badger.DB
	seq      map[string]*badger.Sequence
	DataPath string
}

func InitDb(datapath string, dbopts *chain.DbOption) (*AppDb, error) {
	newdb, err := badger.Open(badger.DefaultOptions(datapath + "_appdb").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return nil, err
	}
	seq := make(map[string]*badger.Sequence)
	appdatalog.Infof("Appdb initialized")
	return &AppDb{Db: newdb, DataPath: datapath, seq: seq}, nil
}

func (appdb *AppDb) GetGroupStatus(groupid string, name string) (string, error) {
	key := fmt.Sprintf("%s%s_%s", STATUS_PREFIX, groupid, name)
	result := ""
	err := appdb.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		b, err := item.ValueCopy(nil)
		result = string(b)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return "", nil
	}

	return result, err
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
			p = append(p, 0xff, 0xff, 0xff, 0xff) // add the postfix 0xffffffff, badger will search the seqid <= 4294967295, it's big enough?
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

func (appdb *AppDb) AddMetaByTrx(blockId string, groupid string, trxs []*quorumpb.Trx) error {
	var err error

	seqkey := SEQ_PREFIX + CNT_PREFIX + GRP_PREFIX + groupid

	keylist := [][]byte{}
	for _, trx := range trxs {
		if trx.Type == quorumpb.TrxType_POST {
			seqid, err := appdb.GetSeqId(seqkey)
			if err != nil {
				return err
			}

			key, err := getKey(fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, groupid), seqid, fmt.Sprintf("%s:%s", trx.SenderPubkey, trx.TrxId))
			if err != nil {
				return err
			}
			keylist = append(keylist, key)
		}
	}

	txn := appdb.Db.NewTransaction(true)
	defer txn.Discard()

	for _, key := range keylist {
		e := badger.NewEntry(key, nil)
		err = txn.SetEntry(e)
		if err != nil {
			return err
		}
	}

	valuename := "HighestBlockId"
	groupLastestBlockidkey := fmt.Sprintf("%s%s_%s", STATUS_PREFIX, groupid, valuename)
	e := badger.NewEntry([]byte(groupLastestBlockidkey), []byte(blockId))
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

func (appdb *AppDb) Close() {
	appdb.Release()
	appdb.Db.Close()
}

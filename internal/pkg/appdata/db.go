package appdata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/orderedcode"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

var appdatalog = logging.Logger("appdata")

const GRP_PREFIX string = "grp_"
const CNT_PREFIX string = "cnt_"
const SDR_PREFIX string = "sdr_"
const SEQ_PREFIX string = "seq_"
const TRX_PREFIX string = "trx_"
const SED_PREFIX string = "sed_"
const STATUS_PREFIX string = "stu_"
const term = "\x00\x01"

type AppDb struct {
	Db       storage.QuorumStorage
	seq      map[string]storage.Sequence
	DataPath string
}
type TrxIdNonce struct {
	TrxId string
	Nonce int64
}

func NewAppDb() *AppDb {
	app := AppDb{}
	app.seq = make(map[string]storage.Sequence)
	return &app
}

func (appdb *AppDb) GetGroupStatus(groupid string, name string) (string, error) {
	key := fmt.Sprintf("%s%s_%s", STATUS_PREFIX, groupid, name)
	exist, err := appdb.Db.IsExist([]byte(key))
	if err != nil {
		return "", err
	}
	if !exist {
		return "", nil
	}

	value, _ := appdb.Db.Get([]byte(key))
	return string(value), err
}

func (appdb *AppDb) GetSeqId(seqkey string) (uint64, error) {
	if appdb.seq[seqkey] == nil {
		seq, err := appdb.Db.GetSequence([]byte(seqkey), 100)
		if err != nil {
			return 0, err
		}
		appdb.seq[seqkey] = seq

	}

	return appdb.seq[seqkey].Next()
}

func (appdb *AppDb) Rebuild(vertag string, chainDb storage.QuorumStorage) error {

	return nil
}

func (appdb *AppDb) GetGroupContentBySenders(groupid string, senders []string, starttrx string, targetnonce int64, num int, reverse bool, starttrxinclude bool) ([]TrxIdNonce, error) {
	prefix := fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, groupid)
	sendermap := make(map[string]bool)
	for _, s := range senders {
		sendermap[s] = true
	}
	trxidsnonce := []TrxIdNonce{}

	p := []byte(prefix)
	if reverse == true {
		p = append(p, 0xff, 0xff, 0xff, 0xff) // add the postfix 0xffffffff, badger will search the seqid <= 4294967295, it's big enough?
	}

	runcollector := false
	if starttrx == "" {
		runcollector = true //no trxid, start collecting from the first item
	}

	err := appdb.Db.PrefixForeachKey(p, []byte(prefix), reverse, func(k []byte, err error) error {
		if err != nil {
			return err
		}
		var trxid, sender string
		var trxnonce int64

		dataidx := bytes.LastIndexByte(k, byte('_'))
		start := dataidx
		seg := 0
		for i, c := range k[dataidx:] {
			if c == ':' {
				if seg == 0 {
					sender = string(k[start+1 : start+i])
					trxid = string(k[start+1+i : start+i+37])
					start = i
					seg = 1
				} else if seg == 1 {
					if len(k)-2 > start+i {
						n := string(k[start+i : len(k)-2])
						trxnonce, err = strconv.ParseInt(n, 10, 64)
					}
				}
			}
		}
		if runcollector == true {
			if len(senders) == 0 || sendermap[sender] == true {
				trxidsnonce = append(trxidsnonce, TrxIdNonce{trxid, trxnonce})
			}
		}
		if trxid == starttrx && runcollector == false { //start collecting after this item
			if targetnonce > 0 {
				if targetnonce == trxnonce {
					runcollector = true
				}
			} else {
				runcollector = true
			}
			if starttrxinclude == true && runcollector == true {
				trxidsnonce = append(trxidsnonce, TrxIdNonce{trxid, trxnonce})
			}
		}
		if len(trxidsnonce) == num {
			// use this to break loop
			return errors.New("OK")
		}
		return nil
	})

	if err != nil && err.Error() == "OK" {
		err = nil
	}

	return trxidsnonce, err
}

func (appdb *AppDb) GetGroupSeed(groupID string) (*quorumpb.GroupSeed, error) {
	key := groupSeedKey(groupID)
	exist, err := appdb.Db.IsExist(key)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}

	value, err := appdb.Db.Get(key)
	if err != nil {
		return nil, err
	}

	var result quorumpb.GroupSeed
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (appdb *AppDb) GetAllGroupSeeds() (map[string]*quorumpb.GroupSeed, error) {
	var seeds map[string]*quorumpb.GroupSeed
	seeds = make(map[string]*quorumpb.GroupSeed)

	key := []byte(SED_PREFIX)
	err := appdb.Db.PrefixForeach(key, func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		var pbSeed quorumpb.GroupSeed
		if err := json.Unmarshal(v, &pbSeed); err != nil {
			return err
		}
		seeds[string(k)] = &pbSeed

		return nil
	})

	return seeds, err
}

func (appdb *AppDb) SetGroupSeed(seed *quorumpb.GroupSeed) error {
	key := groupSeedKey(seed.GroupId)

	value, err := json.Marshal(seed)
	if err != nil {
		return err
	}
	return appdb.Db.Set(key, value)
}

func (appdb *AppDb) DelGroupSeed(groupID string) error {
	key := groupSeedKey(groupID)

	exist, err := appdb.Db.IsExist(key)
	if err != nil {
		return err
	}
	if !exist { // skip
		return nil
	}

	return appdb.Db.Delete(key)
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

			//format:
			//cnt_grp_-6d028f63-d2d0-49aa-9a56-4480ef5a7f2a-_CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg==:b2a3b9aa-bd16-4e80-8497-6d95eddfec52:1
			//cnt_grp_-6d028f63-d2d0-49aa-9a56-4480ef5a7f2a-_CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg==:b2a3b9aa-bd16-4e80-8497-6d95eddfec52
			var tail string
			if trx.Nonce == 0 {
				tail = fmt.Sprintf("%s:%s", trx.SenderPubkey, trx.TrxId)
			} else {
				tail = fmt.Sprintf("%s:%s:%d", trx.SenderPubkey, trx.TrxId, trx.Nonce)
			}
			key, err := getKey(fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, groupid), seqid, tail)
			if err != nil {
				return err
			}
			keylist = append(keylist, key)
		}
	}

	keys := [][]byte{}
	values := [][]byte{}

	for _, key := range keylist {
		keys = append(keys, key)
		values = append(values, nil)
	}

	valuename := "HighestBlockId"
	groupLastestBlockidkey := fmt.Sprintf("%s%s_%s", STATUS_PREFIX, groupid, valuename)
	keys = append(keys, []byte(groupLastestBlockidkey))
	values = append(values, []byte(blockId))

	err = appdb.Db.BatchWrite(keys, values)

	return err
}

func (appdb *AppDb) Release() error {
	for seqkey := range appdb.seq {
		err := appdb.seq[seqkey].Release()
		if err != nil {
			return err
		}
	}
	return nil
}

func (appdb *AppDb) Close() {
	appdb.Release()
	appdb.Db.Close()
}

func groupSeedKey(groupID string) []byte {
	return []byte(fmt.Sprintf("%s%s", SED_PREFIX, groupID))
}

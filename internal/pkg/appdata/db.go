package appdata

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/orderedcode"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var appdatalog = logging.Logger("appdata")

const GRP_PREFIX string = "grp_"
const CNT_PREFIX string = "cnt_"
const SDR_PREFIX string = "sdr_"
const SEQ_PREFIX string = "seq_"
const TRX_PREFIX string = "trx_"
const SED_PREFIX string = "sed_"
const STATUS_PREFIX string = "stu_"

type AppDb struct {
	Db       storage.QuorumStorage
	seq      map[string]storage.Sequence
	DataPath string
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

func (appdb *AppDb) GetGroupContentBySenders(groupid string, senders []string, starttrx string, num int, reverse bool, starttrxinclude bool) (trxidList []string, err error) {
	prefix := fmt.Sprintf("%s%s-%s", CNT_PREFIX, GRP_PREFIX, groupid)
	sendermap := make(map[string]bool)
	for _, s := range senders {
		sendermap[s] = true
	}

	trxids := []string{}
	runcollector := false

	if starttrx == "" {
		runcollector = true //no trxid, start collecting from the first item
	}

	_, err = appdb.Db.PrefixForeachKey([]byte(prefix), []byte(prefix), reverse, func(k []byte, err error) error {
		if err != nil {
			return err
		}

		dataidx := bytes.Index(k[len(prefix):], []byte("_"))
		if dataidx < 0 {
			return nil
		}
		dataidx = len(prefix) + dataidx + 1

		parts := bytes.Split(k[dataidx:], []byte(":"))
		if len(parts) != 2 {
			appdatalog.Warnf("can not get sender and trxid from %s", k[dataidx:])
			return nil
		}

		// Note: sender, trxid contains bytes "\x00\x01"
		sender, trxid := string(parts[0]), string(parts[1])[:36]
		sender = sender[len(sender)-44:]
		if len(sender) != 44 {
			appdatalog.Warnf("key hex: %s prefix: %s invalid sender hex: <%s> len(sender): %d", hex.EncodeToString(k), prefix, hex.EncodeToString([]byte(sender)), len(sender))
			return nil
		}
		if len(trxid) != 36 {
			appdatalog.Warnf("key hex: %s prefix: %s invalid trxid hex: %s", hex.EncodeToString(k), prefix, hex.EncodeToString([]byte(trxid)))
			return nil
		}

		if runcollector {
			if len(senders) == 0 || sendermap[sender] == true {
				trxids = append(trxids, trxid)
			}
		}
		if trxid == starttrx && !runcollector { //start collecting after this item
			runcollector = true
			if starttrxinclude && runcollector {
				trxids = append(trxids, trxid)
			}
		}
		if len(trxids) == num {
			// use this to break loop
			return errors.New("OK")
		}
		return nil
	})

	if err != nil && err.Error() == "OK" {
		err = nil
	}

	return trxids, err
}

func getKey(prefix string, seqid uint64, tailing string) ([]byte, error) {
	return orderedcode.Append(nil, prefix, "-", orderedcode.Infinity, uint64(seqid), "_", tailing)
}

func (appdb *AppDb) AddMetaByTrx(blockId uint64, groupid string, trxs []*quorumpb.Trx) error {
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
			tail = fmt.Sprintf("%s:%s", trx.SenderPubkey, trx.TrxId)
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

	valuename := "Block"
	groupLastestBlockidkey := fmt.Sprintf("%s%s_%s", STATUS_PREFIX, groupid, valuename)
	keys = append(keys, []byte(groupLastestBlockidkey))
	values = append(values, []byte(strconv.FormatUint(blockId, 10)))
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

package appdata

import (
	"fmt"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"log"
	"testing"
	"time"
)

func ifTrxWithNonceSliceEqual(left []TrxIdNonce, right []TrxIdNonce) bool {
	if len(left) != len(right) {
		return false
	}
	for i, vl := range left {
		vr := right[i]
		if vl.TrxId != vr.TrxId || vl.Nonce != vr.Nonce {
			return false
		}
	}
	return true
}

func makemockdb(temppath string, groupid string) (*AppDb, error) {
	name := "testappdb"
	dbname := "appdata"
	tempdir := fmt.Sprintf("%s/%s/%s", temppath, name, dbname)
	log.Printf("tempdir %s", tempdir)

	var err error
	db := storage.QSBadger{}
	err = db.Init(tempdir)
	if err != nil {
		return nil, err
	}
	app := NewAppDb()
	app.Db = &db
	app.DataPath = tempdir
	blockId1 := "bcd19312-c62a-4a20-8fdd-0fd66883d314"
	_ = blockId1
	_ = groupid

	trx1_0 := &quorumpb.Trx{}
	trx1_0.TrxId = "b2a3b9aa-bd16-4e80-8497-6d95eddfec52"
	trx1_0.SenderPubkey = "CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg=="
	trx1_0.GroupId = groupid
	trx1_0.Type = quorumpb.TrxType_POST
	trx1_0.Data = []byte("")
	trx1_0.Version = "1.0.0"
	trx1_0.TimeStamp = time.Now().UnixNano()
	trx1_0.Nonce = int64(0)

	trx1_1 := &quorumpb.Trx{}
	trx1_1.TrxId = "b2a3b9aa-bd16-4e80-8497-6d95eddfec52"
	trx1_1.SenderPubkey = "CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg=="
	trx1_1.GroupId = groupid
	trx1_1.Type = quorumpb.TrxType_POST
	trx1_1.Data = []byte("")
	trx1_1.Version = "1.0.0"
	trx1_1.TimeStamp = time.Now().UnixNano()
	trx1_1.Nonce = int64(1)

	trx1_2 := &quorumpb.Trx{}
	trx1_2.TrxId = "b2a3b9aa-bd16-4e80-8497-6d95eddfec52"
	trx1_2.SenderPubkey = "CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg=="
	trx1_2.GroupId = groupid
	trx1_2.Type = quorumpb.TrxType_POST
	trx1_2.Data = []byte("")
	trx1_2.Version = "1.0.0"
	trx1_2.TimeStamp = time.Now().UnixNano()
	trx1_2.Nonce = int64(2)

	trx1_3 := &quorumpb.Trx{}
	trx1_3.TrxId = "c778c5d0-7fd0-4bdd-867b-cc0bd1d125eb"
	trx1_3.SenderPubkey = "CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg=="
	trx1_3.GroupId = groupid
	trx1_3.Type = quorumpb.TrxType_POST
	trx1_3.Data = []byte("")
	trx1_3.Version = "1.0.0"
	trx1_3.TimeStamp = time.Now().UnixNano()
	trx1_3.Nonce = int64(0)

	trx1_4 := &quorumpb.Trx{}
	trx1_4.TrxId = "0b742adb-69dc-4c81-acea-e7aa19d6e150"
	trx1_4.SenderPubkey = "CAISIQKDY1R5hZ09yG1+i/Kdk8E/KDT8Wm/PrKmgtsdtXFHXEg=="
	trx1_4.GroupId = groupid
	trx1_4.Type = quorumpb.TrxType_POST
	trx1_4.Data = []byte("")
	trx1_4.Version = "1.0.0"
	trx1_4.TimeStamp = time.Now().UnixNano()
	trx1_4.Nonce = int64(0)

	trxs := []*quorumpb.Trx{}
	trxs = append(trxs, trx1_0) //seqid 0
	trxs = append(trxs, trx1_3)
	trxs = append(trxs, trx1_2)
	trxs = append(trxs, trx1_1)
	trxs = append(trxs, trx1_4) //seqid 5

	err = app.AddMetaByTrx(blockId1, groupid, trxs)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func TestAppDb(t *testing.T) {

	groupid := "6d028f63-d2d0-49aa-9a56-4480ef5a7f2a"
	app, err := makemockdb(t.TempDir(), groupid)

	if err != nil {
		t.Errorf("AddMetaByTrx err: %s", err)
	}
	result, _ := app.GetGroupContentBySenders(groupid, []string{}, "b2a3b9aa-bd16-4e80-8497-6d95eddfec52", 2, 20, true, false)
	target := []TrxIdNonce{}
	target = append(target, TrxIdNonce{"c778c5d0-7fd0-4bdd-867b-cc0bd1d125eb", 0})
	target = append(target, TrxIdNonce{"b2a3b9aa-bd16-4e80-8497-6d95eddfec52", 0})

	isequal := ifTrxWithNonceSliceEqual(result, target)
	if isequal == false {
		t.Log("result", result)
		t.Log("target", target)
		t.Errorf("Content result not match with target.")
	}

	result, _ = app.GetGroupContentBySenders(groupid, []string{}, "b2a3b9aa-bd16-4e80-8497-6d95eddfec52", 2, 20, false, false)
	target = []TrxIdNonce{}
	target = append(target, TrxIdNonce{"b2a3b9aa-bd16-4e80-8497-6d95eddfec52", 1})
	target = append(target, TrxIdNonce{"0b742adb-69dc-4c81-acea-e7aa19d6e150", 0})

	isequal = ifTrxWithNonceSliceEqual(result, target)
	if isequal == false {
		t.Log("result", result)
		t.Log("target", target)
		t.Errorf("Content result not match with target.")
	}
}

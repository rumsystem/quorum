package data

import (
	"testing"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var (
	signKeyMap = map[string]string{}

	logger = logging.Logger("data")
)

type TestNonce struct {
	nonce uint64
}

func (tn *TestNonce) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	tn.nonce++
	return tn.nonce, nil
}

func GetGroupItem() *quorumpb.GroupItem {
	var group *quorumpb.GroupItem
	group = &quorumpb.GroupItem{}
	group.GroupId = "7c352591-f237-4b80-81fb-d6347d0380b5"
	group.GroupName = "a_test_group_name"
	group.CipherKey = "71eff58163d557b609a15050a5f7561568eb8bb582156697ba9fc99ca9236582"
	return group
}

func GetKeyStorePubKey(groupid string, path string) (string, string, error) {
	_, err := localcrypto.InitKeystore("defaultkeystore", path)

	if err == nil {
		ks := localcrypto.GetKeystore()
		err = ks.Unlock(signKeyMap, "password")
		signaddr, err := ks.NewKeyWithDefaultPassword(groupid, localcrypto.Sign)
		pubkey, err := ks.GetEncodedPubkey(groupid, localcrypto.Sign)
		return signaddr, pubkey, err
	}
	return "", "", err
}

func TestVerifyTrx(t *testing.T) {
	keystoreDir := t.TempDir()
	tn := &TestNonce{}
	trxFactory := &TrxFactory{}
	groupitem := GetGroupItem()
	trxFactory.Init("1.0.0", groupitem, "default", tn)
	addr, pubkey, err := GetKeyStorePubKey(groupitem.GroupId, keystoreDir)
	logger.Debugf("new eth key addr:%s", addr)
	if err != nil {
		t.Errorf("keystore new key err : %s", err)
	}
	groupitem.UserSignPubkey = pubkey
	obj := &quorumpb.Object{Type: "Note", Name: "test name 1", Content: "test content", Id: "0001"}
	target := &quorumpb.Object{Id: groupitem.GroupId, Type: "Group"}
	postobj := &quorumpb.Activity{Type: "Add", Object: obj, Target: target}

	trx, err := trxFactory.GetPostAnyTrx("", postobj)
	result, err := VerifyTrx(trx)
	if result != true {
		t.Errorf("verify trx sig with pubkey error")
	}
}

func TestVerifyTrxByAddress(t *testing.T) {
	keystoreDir := t.TempDir()
	tn := &TestNonce{}
	trxFactory := &TrxFactory{}
	groupitem := GetGroupItem()
	trxFactory.Init("1.0.0", groupitem, "default", tn)
	addr, _, err := GetKeyStorePubKey(groupitem.GroupId, keystoreDir)
	logger.Debugf("new eth key addr:%s", addr)
	if err != nil {
		t.Errorf("keystore new key err : %s", err)
	}
	groupitem.UserSignPubkey = addr
	obj := &quorumpb.Object{Type: "Note", Name: "test name 1", Content: "test content", Id: "0001"}
	target := &quorumpb.Object{Id: groupitem.GroupId, Type: "Group"}
	postobj := &quorumpb.Activity{Type: "Add", Object: obj, Target: target}

	trx, err := trxFactory.GetPostAnyTrx("", postobj)
	result, err := VerifyTrx(trx)
	if result != true {
		t.Errorf("verify trx sig with pubkey error:%s", err)
	}
}

package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"filippo.io/age"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type (
	trxParam struct {
		TrxId        string
		GroupId      string
		Data         []byte
		TimeStamp    int64
		Version      string
		Expired      int64
		Nonce        int
		SenderPubkey string
	}
)

func nodesdkSendTrx(urls []string, payload *NodeSDKSendTrxItem) (*SendTrxResult, error) {
	urlSuffix := fmt.Sprintf("/api/v1/node/trx/%s", payload.GroupId)
	var result SendTrxResult
	_, _, err := requestNSdk(urls, urlSuffix, "POST", payload, nil, &result, false)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestNodesdkSendTrxToPublicGroup(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-nodesdk-group",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("create group failed: %s, payload: %+v", err, createGroupParam)
	}

	// send trx
	now := time.Now()
	obj := map[string]interface{}{
		"type":    "Note",
		"content": fmt.Sprintf("hello world %d", now.Unix()),
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		t.Errorf("json.Marshal obj failed: %s", err)
	}
	seed, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}
	ciperKey, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		t.Errorf("convert seed.CipherKey failed: %s", err)
	}

	encryptData, err := localcrypto.AesEncrypt(objBytes, ciperKey)
	if err != nil {
		t.Errorf("localcrypto.AesEncrypt failed: %s", err)
	}

	trx := quorumpb.Trx{
		TrxId:        uuid.NewString(),
		GroupId:      group.GroupId,
		Data:         encryptData,
		TimeStamp:    now.UnixNano(),
		Version:      "2.0.0",
		Expired:      now.Add(5 * time.Minute).UnixNano(),
		Nonce:        0, // Note: hardcode
		SenderPubkey: ethPubkey,
	}

	trxWithoutSignBytes, err := proto.Marshal(&trx)
	if err != nil {
		t.Errorf("proto.Marshal trx failed: %s", err)
	}

	hashed := localcrypto.Hash(trxWithoutSignBytes)
	trx.SenderSign, err = ethcrypto.Sign(hashed, ethPrivkey)
	if err != nil {
		t.Errorf("generate eth signature failed: %s", err)
	}

	trxBytes, err := proto.Marshal(&trx)
	if err != nil {
		t.Errorf("proto.Marshal trx failed: %s", err)
	}

	trxJson := struct {
		TrxBytes []byte
	}{
		TrxBytes: trxBytes,
	}
	trxJsonBytes, err := json.Marshal(trxJson)
	if err != nil {
		t.Errorf("json.Marshal trxJson failed: %s", err)
	}

	encTrxJson, err := localcrypto.AesEncrypt(trxJsonBytes, ciperKey)
	if err != nil {
		t.Errorf("json.Marshal trxJson failed: %s", err)
	}

	payload := NodeSDKSendTrxItem{
		GroupId: group.GroupId,
		TrxItem: encTrxJson,
	}

	if _, err := nodesdkSendTrx(urls, &payload); err != nil {
		t.Errorf("send trx via nodesdk rest api failed: %s", err)
	}
}

func TestNodesdkSendTrxToPrivateGroup(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-nodesdk-group",
		ConsensusType:  "poa",
		EncryptionType: "private",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("create group failed: %s, payload: %+v", err, createGroupParam)
	}

	// send trx
	now := time.Now()
	obj := map[string]interface{}{
		"type":    "Note",
		"content": fmt.Sprintf("hello world %d", now.Unix()),
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		t.Errorf("json.Marshal obj failed: %s", err)
	}
	seed, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}
	ciperKey, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		t.Errorf("convert seed.CipherKey failed: %s", err)
	}

	encryptPubkeys, err := getUserEncryptPubKeys(urls, group.GroupId)
	if err != nil {
		t.Errorf("getUserEncryptPubKeys failed: %s", err)
	}

	ageRecipients := []age.Recipient{}
	for _, item := range encryptPubkeys.Keys {
		recipient, err := age.ParseX25519Recipient(item)
		if err != nil {
			t.Errorf("age.ParseX25519Recipient failed: %s", err)
		}
		ageRecipients = append(ageRecipients, recipient)
	}

	encryptData := new(bytes.Buffer)
	if err := localcrypto.AgeEncrypt(ageRecipients, bytes.NewReader(objBytes), encryptData); err != nil {
		t.Errorf("localcrypto.AgeEncrypt failed: %s", err)
	}

	trx := quorumpb.Trx{
		TrxId:        uuid.NewString(),
		GroupId:      group.GroupId,
		Data:         encryptData.Bytes(),
		TimeStamp:    now.UnixNano(),
		Version:      "2.0.0",
		Expired:      now.Add(5 * time.Minute).UnixNano(),
		Nonce:        0, // Note: hardcode
		SenderPubkey: ethPubkey,
	}

	trxWithoutSignBytes, err := proto.Marshal(&trx)
	if err != nil {
		t.Errorf("proto.Marshal trx failed: %s", err)
	}

	hashed := localcrypto.Hash(trxWithoutSignBytes)
	trx.SenderSign, err = ethcrypto.Sign(hashed, ethPrivkey)
	if err != nil {
		t.Errorf("generate eth signature failed: %s", err)
	}

	trxBytes, err := proto.Marshal(&trx)
	if err != nil {
		t.Errorf("proto.Marshal trx failed: %s", err)
	}

	trxJson := struct {
		TrxBytes []byte
	}{
		TrxBytes: trxBytes,
	}
	trxJsonBytes, err := json.Marshal(trxJson)
	if err != nil {
		t.Errorf("json.Marshal trxJson failed: %s", err)
	}

	encTrxJson, err := localcrypto.AesEncrypt(trxJsonBytes, ciperKey)
	if err != nil {
		t.Errorf("json.Marshal trxJson failed: %s", err)
	}

	payload := NodeSDKSendTrxItem{
		GroupId: group.GroupId,
		TrxItem: encTrxJson,
	}

	if _, err := nodesdkSendTrx(urls, &payload); err != nil {
		t.Errorf("send trx via nodesdk rest api failed: %s", err)
	}
}

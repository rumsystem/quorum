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

func nodesdkSendTrx(urls []string, payload *NSdkSendTrxParams) (*SendTrxResult, error) {
	path := fmt.Sprintf("/api/v1/node/%s/trx", payload.GroupId)
	var result SendTrxResult
	_, _, err := requestNSdk(urls, path, "POST", payload, nil, &result, false)
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

	payload := NSdkSendTrxParams{
		GroupId:      group.GroupId,
		TrxId:        trx.TrxId,
		Data:         trx.Data,
		Version:      trx.Version,
		SenderPubkey: trx.SenderPubkey,
		SenderSign:   trx.SenderSign,
		TimeStamp:    trx.TimeStamp,
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
	_, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
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

	payload := NSdkSendTrxParams{
		GroupId:      group.GroupId,
		TrxId:        trx.TrxId,
		Data:         trx.Data,
		Version:      trx.Version,
		SenderPubkey: trx.SenderPubkey,
		SenderSign:   trx.SenderSign,
		TimeStamp:    trx.TimeStamp,
	}

	if _, err := nodesdkSendTrx(urls, &payload); err != nil {
		t.Errorf("send trx via nodesdk rest api failed: %s", err)
	}
}

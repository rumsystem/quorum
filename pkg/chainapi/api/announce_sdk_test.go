package api

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

func announceNSdk(urls []string, payload AnnounceNodeSDKParam) (*handlers.AnnounceResult, error) {
	path := fmt.Sprintf("/api/v1/node/announce/%s", payload.GroupId)
	var result handlers.AnnounceResult
	if _, _, err := requestNSdk(urls, path, "POST", payload, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestAnnounceNSdk(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-sync",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	seed, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}
	ciperKey, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		t.Errorf("convert seed.CipherKey failed: %s", err)
	}

	item := quorumpb.AnnounceItem{
		GroupId:    group.GroupId,
		Type:       quorumpb.AnnounceType_AS_PRODUCER,
		Action:     quorumpb.ActionType_ADD,
		SignPubkey: ethPubkey,
		Result:     quorumpb.ApproveType_ANNOUNCED,
		Memo:       "test announce producer",
	}
	if item.Type == quorumpb.AnnounceType_AS_USER {
		item.EncryptPubkey = ageIdentity.Recipient().String()
	}

	data := item.GroupId + item.SignPubkey + item.EncryptPubkey + item.Type.String()
	hashed := localcrypto.Hash([]byte(data))
	signature, err := ethcrypto.Sign(hashed, ethPrivkey)
	if err != nil {
		t.Errorf("generate eth signature failed: %s", err)
	}

	item.AnnouncerSignature = hex.EncodeToString(signature)
	item.TimeStamp = time.Now().UnixMicro()

	itemBytes, err := proto.Marshal(&item)
	if err != nil {
		t.Errorf("proto.Marshal item failed: %s", err)
	}

	encrypted, err := localcrypto.AesEncrypt(itemBytes, ciperKey)
	if err != nil {
		t.Errorf("localcrypto.AesEncrypt failed: %s", err)
	}

	payload := AnnounceNodeSDKParam{
		GroupId: group.GroupId,
		Req:     encrypted,
	}

	if _, err := announceNSdk(urls, payload); err != nil {
		t.Errorf("announceNSdk failed: %s", err)
	}

	time.Sleep(25 * time.Second)
	producers, err := getChainDataByAnnouncedProducer(urls, AnnGrpProducer{GroupId: group.GroupId}, ciperKey)
	if err != nil {
		t.Errorf("getChainDataByAnnouncedUser failed: %s", err)
	}

	found := false
	for _, item := range producers {
		if item.AnnouncedPubkey == ethPubkey {
			found = true
		}
	}
	if !found {
		t.Errorf("can not find announced user: %s", ethPubkey)
	}
}

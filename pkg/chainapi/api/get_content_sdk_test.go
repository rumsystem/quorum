package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

func getContentNSdk(urls []string, payload *GetGroupCtnReqItem) ([]*quorumpb.Trx, error) {
	urlSuffix := fmt.Sprintf("/api/v1/node/groupctn/%s", payload.GroupId)
	var result []*quorumpb.Trx
	if _, _, err := requestNSdk(urls, urlSuffix, "POST", payload, nil, &result, true); err != nil {
		return nil, err
	}

	return result, nil
}

func TestGetContentNSdk(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-get-cnt",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	post := PostGroupParam{
		GroupID: group.GroupId,
		Data: map[string]interface{}{
			"Type":    "Note",
			"Content": "hello world ...",
		},
	}
	if _, err := postToGroup(peerapi, post); err != nil {
		t.Errorf("post to group failed: %s", err)
	}

	time.Sleep(10 * time.Second)

	seed, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}
	ciperKey, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		t.Errorf("convert seed.CipherKey failed: %s", err)
	}

	reverse := "true"
	if rand.Intn(20) > 10 {
		reverse = "false"
	}
	param := GetGroupCtnItem{
		Req: GetGroupCtnPrarms{
			GroupId:         group.GroupId,
			Num:             20,
			Reverse:         reverse,
			IncludeStartTrx: "false",
		},
	}

	paramBytes, err := json.Marshal(param)
	if err != nil {
		t.Errorf("json.Marshal param failed: %s", err)
	}

	encrypted, err := localcrypto.AesEncrypt(paramBytes, ciperKey)
	if err != nil {
		t.Errorf("localcrypto.AesEncrypt failed: %s", err)
	}

	payload := GetGroupCtnReqItem{
		GroupId: group.GroupId,
		Req:     encrypted,
	}
	if _, err := getContentNSdk(urls, &payload); err != nil {
		t.Errorf("get content via sdk api failed: %s", err)
	}
}

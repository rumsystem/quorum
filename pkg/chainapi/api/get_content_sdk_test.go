package api

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

func getContentNSdk(urls []string, payload *handlers.GetGroupCtnPrarms) ([]*quorumpb.Trx, error) {
	path := fmt.Sprintf("/api/v1/node/%s/groupctn", payload.GroupId)
	var result []*quorumpb.Trx
	if _, _, err := requestNSdk(urls, path, "GET", payload, nil, &result, true); err != nil {
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

	_, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}

	reverse := true
	if rand.Intn(20) > 10 {
		reverse = false
	}
	payload := handlers.GetGroupCtnPrarms{
		GroupId:         group.GroupId,
		Num:             20,
		Reverse:         reverse,
		IncludeStartTrx: false,
	}

	if _, err := getContentNSdk(urls, &payload); err != nil {
		t.Errorf("get content via sdk api failed: %s", err)
	}
}

package api

import (
	"testing"

	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func TestPostToGroup(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-post",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	// join group
	joinGroupParam := handlers.JoinGroupParamV2{
		Seed: group.Seed,
	}
	if _, err := joinGroup(peerapi2, joinGroupParam); err != nil {
		t.Errorf("joinGroup failed: %s, payload: %+v", err, joinGroupParam)
	}

	// post to group
	postGroupParam := PostGroupParam{
		Data: map[string]interface{}{
			"type":    "Note",
			"content": "Hello World",
			"name":    "post to group testing",
		},
		GroupID: group.GroupId,
	}

	postResult, err := postToGroup(peerapi, postGroupParam)
	if err != nil {
		t.Errorf("postToGroup failed: %s, payload: %+v", err, postGroupParam)
	}
	if postResult.TrxId == "" {
		t.Errorf("postToGroup failed: TrxId is empty")
	}
}

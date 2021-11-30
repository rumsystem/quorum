package api

import (
	"testing"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func TestPostToGroup(t *testing.T) {
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
	joinGroupParam := handlers.GroupSeed{
		GenesisBlock:   group.GenesisBlock,
		GroupId:        group.GroupId,
		GroupName:      group.GroupName,
		OwnerPubkey:    group.OwnerPubkey,
		ConsensusType:  group.ConsensusType,
		EncryptionType: group.EncryptionType,
		CipherKey:      group.CipherKey,
		AppKey:         group.AppKey,
		Signature:      group.Signature,
	}
	if _, err := joinGroup(peerapi2, joinGroupParam); err != nil {
		t.Errorf("joinGroup failed: %s, payload: %+v", err, joinGroupParam)
	}

	// post to group
	postGroupParam := PostGroupParam{
		Type: "Add",
		Object: PostObject{
			Type:    "Note",
			Content: "Hello World",
			Name:    "post to group testing",
		},
		Target: PostTarget{
			Type: "Group",
			ID:   group.GroupId,
		},
	}

	postResult, err := postToGroup(peerapi, postGroupParam)
	if err != nil {
		t.Errorf("postToGroup failed: %s, payload: %+v", err, postGroupParam)
	}
	if postResult.TrxId == "" {
		t.Errorf("postToGroup failed: TrxId is empty")
	}
}

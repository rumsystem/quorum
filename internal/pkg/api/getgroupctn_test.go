package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func TestGetGroupContent(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-group-content",
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
	content := fmt.Sprintf("%s hello world", RandString(4))
	name := fmt.Sprintf("%s post to group testing", RandString(4))
	postGroupParam := PostGroupParam{
		Type: "Add",
		Object: PostObject{
			Type:    "Note",
			Content: content,
			Name:    name,
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

	// FIXME
	time.Sleep(time.Second * 25)

	// check peerapi received content
	for _, api := range []string{peerapi, peerapi2} {
		receivedContent, err := isReceivedGroupContent(api, group.GroupId, postResult.TrxId)
		if err != nil {
			t.Errorf("isReceivedGroupContent failed: %s", err)
		}
		if !receivedContent {
			t.Errorf("isReceivedGroupContent failed: content not received")
		}
	}
}

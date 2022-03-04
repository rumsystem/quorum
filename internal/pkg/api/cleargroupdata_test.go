package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func clearGroup(api string, payload handlers.ClearGroupDataParam) (*handlers.ClearGroupDataResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := string(payloadBytes[:])
	urlSuffix := fmt.Sprintf("/api/v1/group/clear")
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.ClearGroupDataResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestClearGroup(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-get-trx",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
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

	if _, err := postToGroup(peerapi, postGroupParam); err != nil {
		t.Errorf("postToGroup failed: %s, payload: %+v", err, postGroupParam)
	}

	if _, err := clearGroup(peerapi, handlers.ClearGroupDataParam{GroupId: group.GroupId}); err != nil {
		t.Fatalf("clearGroup failed: %s", err)
	}
}

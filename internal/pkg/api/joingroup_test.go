package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func joinGroup(api string, payload handlers.GroupSeed) (*JoinGroupResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		e := fmt.Errorf("json.Marshal failed: %s, joinGroupParam: %+v", err, payload)
		return nil, e
	}

	payloadStr := string(payloadByte[:])
	urlPath := "/api/v1/group/join"
	_, resp, err := testnode.RequestAPI(api, urlPath, "POST", payloadStr)
	if err != nil {
		e := fmt.Errorf("request %s failed: %s, payload: %s", urlPath, err, payloadStr)
		return nil, e
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result JoinGroupResult
	if err := json.Unmarshal(resp, &result); err != nil {
		e := fmt.Errorf("json.Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		e := fmt.Errorf("join group response body invalid: %s, response: %+v", err, result)
		return nil, e
	}

	return &result, nil
}

func TestJoinGroup(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-join-group",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("create group failed: %s, payload: %+v", err, createGroupParam)
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

	// check if it in group list
	inGroup, err := isInGroup(peerapi2, joinGroupParam.GroupId)
	if err != nil {
		t.Errorf("isInGroup failed: %s", err)
	}
	if !inGroup {
		t.Errorf("joined in group, but it is not in group list")
	}
}

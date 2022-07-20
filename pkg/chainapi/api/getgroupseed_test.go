package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func getGroupSeed(api string, groupID string) (*handlers.GetGroupSeedResult, error) {
	path := fmt.Sprintf("/api/v1/group/%s/seed", groupID)
	_, resp, err := testnode.RequestAPI(api, path, "GET", "")
	if err != nil {
		return nil, fmt.Errorf("get group seed failed: %s", err)
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var seed handlers.GetGroupSeedResult
	if err := json.Unmarshal(resp, &seed); err != nil {
		e := fmt.Errorf("response Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	validate := validator.New()
	if err := validate.Struct(seed); err != nil {
		return nil, err
	}

	seedObj, _, err := handlers.UrlToGroupSeed(seed.Seed)
	if err != nil {
		return nil, err
	}
	if seedObj.GroupId != groupID {
		return nil, fmt.Errorf("group id not match, expect: %s, actual: %s", groupID, seedObj.GroupId)
	}

	return &seed, nil
}

func TestGetGroupSeed(t *testing.T) {
	payload := handlers.CreateGroupParam{
		AppKey:         "default",
		ConsensusType:  "poa",
		EncryptionType: "public",
		GroupName:      fmt.Sprintf("test-seed-%d", time.Now().Unix()),
	}

	group, err := createGroup(peerapi, payload)
	if err != nil {
		t.Fatalf("create group failed: %s", err)
	}

	if _, err := getGroupSeed(peerapi, group.GroupId); err != nil {
		t.Fatalf("get group seed failed: %s", err)
	}
}

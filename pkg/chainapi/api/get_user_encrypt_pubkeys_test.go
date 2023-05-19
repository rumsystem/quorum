package api

import (
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func getUserEncryptPubKeys(urls []string, groupID string) (*GetUserEncryptPubKeysResult, error) {
	path := fmt.Sprintf("/api/v1/node/%s/encryptpubkeys", groupID)
	var result GetUserEncryptPubKeysResult
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, false); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestGetUserEncryptPubKeys(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-get-pubkeys",
		ConsensusType:  "poa",
		EncryptionType: "private",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	_, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}

	if _, err := getUserEncryptPubKeys(urls, group.GroupId); err != nil {
		t.Errorf("getUserEncryptPubKeys failed: %s", err)
	}
}

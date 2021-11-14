package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/testnode"
)

func getBlockedUsers(api, groupID string) ([]*DeniedUserListItem, error) {
	urlSuffix := fmt.Sprintf("/api/v1/group/%s/deniedlist", groupID)
	resp, err := testnode.RequestAPI(api, urlSuffix, "GET", "")
	if err != nil {
		return nil, err
	}

	var deniedUsers []*DeniedUserListItem
	if err := json.Unmarshal(resp, &deniedUsers); err != nil {
		return nil, err
	}

	for _, user := range deniedUsers {
		validate := validator.New()
		if err := validate.Struct(user); err != nil {
			return nil, err
		}
	}

	return deniedUsers, nil
}

func TestGroupBlockedUersIsNone(t *testing.T) {
	// creae group
	createGroupParam := CreateGroupParam{
		GroupName:      "test-blked-user",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s", err)
	}

	// get blocked users
	deniedUsers, err := getBlockedUsers(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getBlockedUsers failed: %s", err)
	}

	if deniedUsers != nil {
		t.Fatalf("getBlockedUsers should be nil")
	}
}

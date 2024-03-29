package api

import (
	"testing"

	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func TestLeaveGroup(t *testing.T) {
	t.Skip()
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-leave-group",
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

	// list my group
	inGroup, err := isInGroup(peerapi2, group.GroupId)
	if err != nil {
		t.Errorf("isInGroup failed: %s", err)
	}
	if !inGroup {
		t.Errorf("try to joined in group, but it not in group list")
	}

	// leave group
	leaveGroupParam := handlers.LeaveGroupParam{
		GroupId: group.GroupId,
	}
	_, err = leaveGroup(peerapi2, leaveGroupParam)
	if err != nil {
		t.Errorf("leaveGroup failed: %s", err)
	}

	// clear group, NOTE: need stop sync?
	if _, err := clearGroup(peerapi2, handlers.ClearGroupDataParam{GroupId: group.GroupId}); err != nil {
		t.Fatalf("clearGroup failed: %s", err)
	}

	// check group list
	inGroup, err = isInGroup(peerapi2, group.GroupId)
	if err != nil {
		t.Errorf("isInGroup failed: %s", err)
	}
	if inGroup {
		t.Errorf("left group, but it still in group list")
	}
}

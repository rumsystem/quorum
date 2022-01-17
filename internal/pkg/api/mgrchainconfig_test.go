package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func mgrGroupBlockedList(api string, payload handlers.DenyListParam) (*handlers.DenyUserResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := string(payloadBytes[:])
	resp, err := testnode.RequestAPI(api, "/api/v1/group/deniedlist", "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.DenyUserResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	if result.Action != payload.Action {
		e := fmt.Errorf("result.Action should be %s, but got %s", payload.Action, result.Action)
		return nil, e
	}

	if result.GroupId != payload.GroupId {
		e := fmt.Errorf("result.GroupId should be %s, but got %s", payload.GroupId, result.GroupId)
		return nil, e
	}

	if result.PeerId != payload.PeerId {
		e := fmt.Errorf("result.PeerId should be %s, but got %s", payload.PeerId, result.PeerId)
		return nil, e
	}

	if result.Memo != payload.Memo {
		e := fmt.Errorf("result.Memo should be %s, but got %s", payload.Memo, result.Memo)
		return nil, e
	}

	return &result, nil
}

func TestMgrGroupBlockedList(t *testing.T) {
	appKey := "default"
	consensusType := "poa"
	encryptionType := "public"
	groupName := fmt.Sprintf("%s-%d", encryptionType, time.Now().Unix())
	payload := handlers.CreateGroupParam{
		AppKey:         appKey,
		ConsensusType:  consensusType,
		EncryptionType: encryptionType,
		GroupName:      groupName,
	}

	group, err := createGroup(peerapi, payload)
	if err != nil {
		t.Fatalf("create group failed: %s", err)
	}

	node2, err := getNodeInfo(peerapi2)
	if err != nil {
		t.Fatalf("getNodeInfo failed: %s", err)
	}

	blockedUsers, err := getBlockedUsers(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getBlockedUsers failed: %s", err)
	}
	if blockedUsers != nil {
		t.Fatalf("blockedUsers should be nil")
	}

	// add blocked user
	param := handlers.DenyListParam{
		Action:  "add",
		PeerId:  node2.NodeID,
		GroupId: group.GroupId,
		Memo:    "test add",
	}
	if _, err := mgrGroupBlockedList(peerapi, param); err != nil {
		t.Fatalf("MgrGroupBlockedList failed: %s, payload: %+v", err, param)
	}

	// get blockedUsers
	time.Sleep(time.Second * 15)
	blockedUsers, err = getBlockedUsers(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getBlockedUsers failed: %s", err)
	}
	if blockedUsers == nil || len(blockedUsers) != 1 {
		t.Errorf("blockedUsers should not be nil or empty")
	}

	// delete blocked user
	param = handlers.DenyListParam{
		Action:  "del",
		PeerId:  node2.NodeID,
		GroupId: group.GroupId,
		Memo:    "test delete",
	}
	if _, err := mgrGroupBlockedList(peerapi, param); err != nil {
		t.Fatalf("MgrGroupBlockedList failed: %s, payload: %+v", err, param)
	}

	// get blockedUsers
	time.Sleep(time.Second * 15)
	blockedUsers, err = getBlockedUsers(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getBlockedUsers failed: %s", err)
	}

	if blockedUsers != nil {
		t.Fatalf("blockedUsers should be nil")
	}
}

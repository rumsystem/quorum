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

func getResponseError(resp []byte) error {
	var data1 map[string]interface{}
	var data2 [](map[string]interface{})
	err1 := json.Unmarshal(resp, &data1)
	err2 := json.Unmarshal(resp, &data2)
	if err1 != nil && err2 != nil {
		return fmt.Errorf("err1: %s, err2: %s", err1, err2)
	}

	if data1 != nil {
		if _, found := data1["error"]; found {
			return fmt.Errorf("%s", data1["error"])
		}
	}

	return nil
}

func createGroup(api string, payload handlers.CreateGroupParam) (*handlers.GroupSeed, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		e := fmt.Errorf("json.Marshal failed, payload: %+v error: %s", payload, err)
		return nil, e
	}
	payloadStr := string(payloadByte[:])

	_, resp, err := testnode.RequestAPI(api, "/api/v1/group", "POST", payloadStr)
	if err != nil {
		e := fmt.Errorf("create group %s failed: %s", payload.GroupName, err)
		return nil, e
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var group handlers.GroupSeed
	if err := json.Unmarshal(resp, &group); err != nil {
		e := fmt.Errorf("response Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	validate := validator.New()
	if err := validate.Struct(group); err != nil {
		return nil, err
	}

	if group.GroupId == "" {
		e := fmt.Errorf("create group failed, `GroupId` should not been empty")
		return nil, e
	}

	if group.GroupName != payload.GroupName {
		e := fmt.Errorf("request payload GroupName: %s != response GroupName: %s", payload.GroupName, group.GroupName)
		return nil, e
	}

	if group.AppKey != payload.AppKey {
		e := fmt.Errorf("request payload AppKey: %s != response AppKey: %s", payload.AppKey, group.AppKey)
		return nil, e
	}

	if group.ConsensusType != payload.ConsensusType {
		e := fmt.Errorf("request payload ConsensusType: %s != response ConsensusType: %s", payload.ConsensusType, group.ConsensusType)
		return nil, e
	}

	if group.EncryptionType != payload.EncryptionType {
		e := fmt.Errorf("request payload EncryptionType: %s != response EncryptionType: %s", payload.EncryptionType, group.EncryptionType)
		return nil, e
	}

	// check group[genesis_block]
	respGenesisBlock := group.GenesisBlock
	if group.GroupId != respGenesisBlock.GroupId {
		e := fmt.Errorf("response[group_id]: %s != response[genesis_block][GroupId]: %s", group.GroupId, respGenesisBlock.GroupId)
		return nil, e
	}

	return &group, nil
}

func getGroups(api string) (*GroupInfoList, error) {
	_, resp, err := testnode.RequestAPI(api, "/api/v1/groups", "GET", "")
	if err != nil {
		e := fmt.Errorf("get groups failed: %s", err)
		return nil, e
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var groups GroupInfoList
	if err := json.Unmarshal(resp, &groups); err != nil {
		e := fmt.Errorf("response Unmarshal error: %s", err)
		return nil, e
	}

	validate := validator.New()
	if err := validate.Struct(groups); err != nil {
		e := fmt.Errorf("get group response body invalid: %s, response: %+v", err, groups)
		return nil, e
	}

	return &groups, nil
}

func leaveGroup(api string, payload handlers.LeaveGroupParam) (*handlers.LeaveGroupResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadStr := string(payloadByte[:])
	_, resp, err := testnode.RequestAPI(api, "/api/v1/group/leave", "POST", payloadStr)
	if err != nil {
		return nil, err
	}
	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.LeaveGroupResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	if result.GroupId != payload.GroupId {
		e := fmt.Errorf("response group id != request group id: %s != %s", result.GroupId, payload.GroupId)
		return nil, e
	}

	return &result, nil
}

func isInGroup(api string, groupID string) (bool, error) {
	groups, err := getGroups(api)
	if err != nil {
		return false, err
	}

	for _, g := range groups.GroupInfos {
		if g.GroupId == groupID {
			return true, nil
		}
	}

	return false, nil
}

type PostObject struct {
	Type    string `json:"type" validate:"required"`
	Content string `json:"content" validate:"required"`
	Name    string `json:"name" validate:"required"`
}

type PostTarget struct {
	Type string `json:"type" validate:"required"`
	ID   string `json:"id" validate:"required"`
}

type PostGroupParam struct {
	Type   string     `json:"type" validate:"required"`
	Object PostObject `json:"object" validate:"required"`
	Target PostTarget `json:"target" validate:"required"`
}

func postToGroup(api string, payload PostGroupParam) (*handlers.TrxResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadStr := string(payloadByte[:])
	_, resp, err := testnode.RequestAPI(api, "/api/v1/group/content", "POST", payloadStr)
	if err != nil {
		return nil, err
	}
	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.TrxResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

type ContentInnerStruct struct {
	Type    string `json:"type" validate:"required"`
	Content string `json:"content" validate:"required"`
	Name    string `json:"name" validate:"required"`
}

type GroupContentItem struct {
	TrxId     string             `json:"TrxId" validate:"required"`
	Publisher string             `json:"Publisher" validate:"required"`
	Content   ContentInnerStruct `json:"Content" validate:"required"`
	TypeUrl   string             `json:"TypeUrl" validate:"required"`
	TimeStamp int64              `json:"TimeStamp" validate:"required"`
}

func getGroupContent(api string, groupID string) ([]GroupContentItem, error) {
	urlSuffix := fmt.Sprintf("/api/v1/group/%s/content", groupID)
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "GET", "")
	if err != nil {
		return nil, err
	}
	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result []GroupContentItem
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	for _, item := range result {
		validate := validator.New()
		if err := validate.Struct(item); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func isReceivedGroupContent(api string, groupID string, trxID string) (bool, error) {
	contents, err := getGroupContent(api, groupID)
	if err != nil {
		return false, err
	}

	for _, content := range contents {
		if content.TrxId == trxID {
			return true, nil
		}
	}

	return false, nil
}

/*
func TestGetNullGroups(t *testing.T) {
	groups, err := getGroups(peerapi)
	if err != nil {
		t.Errorf("getGroups failed: %s", err)
	}

	groupInfos := groups.GroupInfos
	if groupInfos != nil {
		t.Errorf("it should none groups, but groups is: %+v", groups)
	}
}
*/

func TestCreateAndGetGroups(t *testing.T) {
	appKey := "default"
	consensusType := "poa"
	encryptionTypes := []string{"public", "private"}

	for _, encryptionType := range encryptionTypes {
		groupName := fmt.Sprintf("%s-%d", encryptionType, time.Now().Unix())
		payload := handlers.CreateGroupParam{
			AppKey:         appKey,
			ConsensusType:  consensusType,
			EncryptionType: encryptionType,
			GroupName:      groupName,
		}

		if _, err := createGroup(peerapi, payload); err != nil {
			t.Errorf("create group failed: %s", err)
		}

		groups, err := getGroups(peerapi)
		if err != nil {
			t.Errorf("getGroups failed: %s", err)
		}

		if groups.GroupInfos == nil {
			t.Error("it should least one group, but groups is null")
		}
	}
}

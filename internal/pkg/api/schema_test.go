package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func addOrRemoveSchema(api string, payload SchemaParam) (*SchemaResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadStr := string(payloadBytes)

	urlSuffix := "/api/v1/group/schema"
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result SchemaResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	if result.GroupId != payload.GroupId {
		e := fmt.Errorf("result.GroupId %s != payload.GroupId %s", result.GroupId, payload.GroupId)
		return nil, e
	}
	resultAction := strings.ToUpper(payload.Action)
	if result.Action != resultAction {
		e := fmt.Errorf("result.Action %s != %s", result.Action, resultAction)
		return nil, e
	}
	if result.SchemaRule != payload.Rule {
		e := fmt.Errorf("result.SchemaRule %s != payload.Rule %s", result.SchemaRule, payload.Rule)
		return nil, e
	}
	if result.SchemaType != payload.Type {
		e := fmt.Errorf("result.SchemaType %s != payload.Type %s", result.SchemaType, payload.Type)
		return nil, e
	}

	return &result, nil
}

func getSchemaList(api string, groupID string) ([]SchemaListItem, error) {
	urlSuffix := fmt.Sprintf("/api/v1/group/%s/app/schema", groupID)
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "GET", "")
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result []SchemaListItem
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

func TestGetAddRemoveSchema(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-schema",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	// get schema
	schemaList, err := getSchemaList(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getSchemaList failed: %s, groupID: %s", err, group.GroupId)
	}
	if len(schemaList) != 0 {
		t.Fatalf("getSchemaList failed: len(schemaList) %d != 0", len(schemaList))
	}

	// add schema
	_type := "schema_type"
	rule := "test-schema"
	memo := "memo"
	schemaParam := SchemaParam{
		GroupId: group.GroupId,
		Action:  "add",
		Type:    _type,
		Rule:    rule,
		Memo:    memo,
	}
	if _, err := addOrRemoveSchema(peerapi, schemaParam); err != nil {
		t.Fatalf("addOrRemoveSchema failed: %s, payload: %+v", err, schemaParam)
	}

	// get schema
	time.Sleep(time.Second * 15)
	schemaList, err = getSchemaList(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getSchemaList failed: %s, groupID: %s", err, group.GroupId)
	}
	if len(schemaList) != 1 {
		t.Fatalf("getSchemaList failed: len(schemaList) %d != 1", len(schemaList))
	}

	// remove schema
	removeSchemaParam := SchemaParam{
		GroupId: group.GroupId,
		Action:  "remove",
		Type:    _type,
		Rule:    rule,
		Memo:    memo,
	}
	if _, err := addOrRemoveSchema(peerapi, removeSchemaParam); err != nil {
		t.Fatalf("addOrRemoveSchema failed: %s, payload: %+v", err, removeSchemaParam)
	}

	// get schema
	time.Sleep(time.Second * 15)
	schemaList, err = getSchemaList(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getSchemaList failed: %s, groupID: %s", err, group.GroupId)
	}
	if len(schemaList) != 0 {
		t.Fatalf("getSchemaList failed: len(schemaList) %d != 0", len(schemaList))
	}
}

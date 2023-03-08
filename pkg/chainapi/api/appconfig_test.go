package api

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func updateAppConfig(api string, payload handlers.AppConfigParam) (*handlers.AppConfigResult, error) {
	path := "/api/v1/group/appconfig"
	var result handlers.AppConfigResult
	if _, _, err := requestAPI(api, path, "POST", payload, nil, &result, false); err != nil {
		return nil, err
	}
	return &result, nil
}

func getAppConfigKeyList(api string, groupId string) ([]*handlers.AppConfigKeyListItem, error) {
	path := fmt.Sprintf("/api/v1/group/%s/appconfig/keylist", groupId)
	var result []*handlers.AppConfigKeyListItem
	if _, _, err := requestAPI(api, path, "GET", nil, nil, &result, true); err != nil {
		return nil, err
	}

	for _, item := range result {
		validate := validator.New()
		if err := validate.Struct(item); err != nil {
			e := fmt.Errorf("validate.Struct failed: %s, struct: %+v", err, item)
			return nil, e
		}
	}

	return result, nil
}

func getAppConfigByName(api string, groupId string, name string) (*handlers.AppConfigKeyItem, error) {
	path := fmt.Sprintf("/api/v1/group/%s/appconfig/%s", groupId, name)
	var result handlers.AppConfigKeyItem
	if _, _, err := requestAPI(api, path, "GET", nil, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestUpdateAndGetAppConfig(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-appconfig",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	for _, _type := range []string{"bool", "int", "string"} {
		value := ""
		if _type == "bool" {
			value = "false"
		} else if _type == "int" {
			value = "100"
		} else if _type == "string" {
			value = "hello"
		}

		name := fmt.Sprintf("test-add-%s", _type)
		params := handlers.AppConfigParam{
			Action:  "add",
			GroupId: group.GroupId,
			Type:    _type,
			Name:    name,
			Value:   value,
			Memo:    fmt.Sprintf("test add type %s", _type),
		}

		if _, err := updateAppConfig(peerapi, params); err != nil {
			t.Errorf("update appconfig failed: %s", err)
		}

		time.Sleep(15 * time.Second)

		keylist, err := getAppConfigKeyList(peerapi, group.GroupId)
		if err != nil {
			t.Errorf("get appconfig keylist failed: %s", err)
		}

		found := false
		for _, item := range keylist {
			if item.Name == name && item.Type == strings.ToUpper(_type) {
				found = true
			}
		}
		if !found {
			t.Errorf("can not find appconfig keylist, name: %s type: %s", name, _type)
		}

		appconfig, err := getAppConfigByName(peerapi, group.GroupId, name)
		if err != nil {
			t.Errorf("get appconfig by name %s failed: %s", name, err)
		}
		if appconfig.Name != name || appconfig.Type != strings.ToUpper(_type) {
			t.Errorf("get appconfig by name failed: %s, except name: %s type: %s, actual name: %s type: %s", err, name, _type, appconfig.Name, appconfig.Type)
		}
	}
}

package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func getChainDataByAuthType(urls []string, payload GetNSdkAuthTypeParams) (*handlers.TrxAuthItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/auth/by/%s", payload.GroupId, payload.TrxType)
	var result handlers.TrxAuthItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func getChainDataByAuthAllowList(urls []string, payload GetNSdkAllowListParams) ([]handlers.ChainSendTrxRuleListItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/auth/alwlist", payload.GroupId)
	var result []handlers.ChainSendTrxRuleListItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, true); err != nil {
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

func getChainDataByAuthDenyList(urls []string, payload GetNSdkDenyListParams) ([]*handlers.ChainSendTrxRuleListItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/auth/denylist", payload.GroupId)
	var result []*handlers.ChainSendTrxRuleListItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, true); err != nil {
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

func getChainDataByAppConfigKeyList(urls []string, payload GetNSdkAppconfigKeylistParams) ([]*handlers.AppConfigKeyListItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/appconfig/keylist", payload.GroupId)
	var result []*handlers.AppConfigKeyListItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, true); err != nil {
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

func getChainDataByAppConfigItemByKey(urls []string, payload GetNSdkAppconfigByKeyParams) (*handlers.AppConfigKeyItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/appconfig/by/%s", payload.GroupId, payload.Key)
	var result handlers.AppConfigKeyItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func getChainDataByAnnouncedProducer(urls []string, payload GetNSdkAnnouncedProducerParams) ([]*handlers.AnnouncedProducerListItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/announced/producer", payload.GroupId)
	var result []*handlers.AnnouncedProducerListItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, true); err != nil {
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

func getChainDataByAnnouncedUser(urls []string, payload GetNSdkAnnouncedUserParams) ([]*handlers.AnnouncedUserListItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/announced/user", payload.GroupId)
	var result []*handlers.AnnouncedUserListItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, true); err != nil {
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

func getChainDataByGroupProducer(urls []string, payload GetNSdkGroupProducersParams) ([]*handlers.ProducerListItem, error) {
	path := fmt.Sprintf("/api/v1/node/%s/producers", payload.GroupId)
	var result []*handlers.ProducerListItem
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, true); err != nil {
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

func getChainDataByGroupInfo(urls []string, payload GetNSdkGroupInfoParams) (*GrpInfoNodeSDK, error) {
	path := fmt.Sprintf("/api/v1/node/%s/info", payload.GroupId)
	var result GrpInfoNodeSDK
	if _, _, err := requestNSdk(urls, path, "GET", nil, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestGetChainDataNSdk(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:       "test-chain-data",
		ConsensusType:   "poa",
		EncryptionType:  "public",
		AppKey:          "default",
		IncludeChainUrl: true,
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	_, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}

	if _, err := getChainDataByGroupInfo(urls, GetNSdkGroupInfoParams{GroupId: group.GroupId}); err != nil {
		t.Errorf("get group info from chain data failed: %s", err)
	}

	// update chain, set `POST` to `follow_alw_list`
	_trxType := "POST"
	_trxAuthMode := "follow_alw_list"
	authParams := handlers.TrxAuthModeParams{
		TrxType:     _trxType,
		TrxAuthMode: _trxAuthMode,
	}
	authBytes, err := json.Marshal(authParams)
	if err != nil {
		t.Errorf("json.Marshal failed: %s", err)
	}

	payload := handlers.ChainConfigParams{
		GroupId: group.GroupId,
		Type:    "set_trx_auth_mode",
		Config:  string(authBytes),
		Memo:    fmt.Sprintf("memo-%d", time.Now().Unix()),
	}

	if _, err := updateChainConfig(peerapi, payload); err != nil {
		t.Errorf("update chain config with payload: %+v failed: %s", payload, err)
	}
	time.Sleep(25 * time.Second)
	authTypeResult, err := getChainDataByAuthType(urls, GetNSdkAuthTypeParams{GroupId: group.GroupId, TrxType: _trxType})
	if err != nil {
		t.Errorf("getChainDataByAuthType failed: %s", err)
	}

	if authTypeResult.TrxType != strings.ToUpper(_trxType) || authTypeResult.AuthType != strings.ToUpper(_trxAuthMode) {
		t.Errorf("getChainDataByAuthType failed, TrxType expect: %s actual: %s, AuthType expect: %s actual: %s", strings.ToUpper(_trxType), authTypeResult.TrxType, strings.ToUpper(_trxAuthMode), authTypeResult.AuthType)
	}

	// update `POST` `follow_alw_list`
	config := handlers.ChainSendTrxRuleListItemParams{
		Action:  "add",
		Pubkey:  ethPubkey,
		TrxType: []string{"POST"},
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		t.Errorf("json.Marshal(%+v) failed: %s", config, err)
	}

	payload = handlers.ChainConfigParams{
		GroupId: group.GroupId,
		Type:    "upd_alw_list",
		Config:  string(configBytes),
		Memo:    fmt.Sprintf("memo-%d", time.Now().Unix()),
	}
	if _, err := updateChainConfig(peerapi, payload); err != nil {
		t.Errorf("update chain config with payload: %+v failed: %s", payload, err)
	}
	time.Sleep(25 * time.Second)

	if _, err := getChainDataByAuthAllowList(urls, GetNSdkAllowListParams{GroupId: group.GroupId}); err != nil {
		t.Errorf("getChainDataByAuthAllowList failed: %s", err)
	}
}

func TestGetChainDataAppConfigNSdk(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:       "test-chain-data",
		ConsensusType:   "poa",
		EncryptionType:  "public",
		AppKey:          "default",
		IncludeChainUrl: true,
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	_, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}

	// set appconfig
	name := "test-add-int"
	_type := "int"
	params := handlers.AppConfigParam{
		Action:  "add",
		GroupId: group.GroupId,
		Type:    _type,
		Name:    name,
		Value:   "100",
		Memo:    "test add int type",
	}

	if _, err := updateAppConfig(peerapi, params); err != nil {
		t.Errorf("update appconfig failed: %s", err)
	}

	time.Sleep(25 * time.Second)

	keylist, err := getChainDataByAppConfigKeyList(urls, GetNSdkAppconfigKeylistParams{GroupId: group.GroupId})
	if err != nil {
		t.Errorf("get appconfig keylist by sdk api failed: %s", err)
	}

	found := false
	for _, item := range keylist {
		if item.Name == name && item.Type == strings.ToUpper(_type) {
			found = true
		}
	}
	if !found {
		t.Errorf("can not find appconfig by name: %s", name)
	}

	appconfig, err := getChainDataByAppConfigItemByKey(
		urls, GetNSdkAppconfigByKeyParams{GroupId: group.GroupId, Key: name},
	)
	if err != nil {
		t.Errorf("get appconfig by name: %s failed: %s", err, name)
	}

	if appconfig.Name != name || appconfig.Type != strings.ToUpper(_type) {
		t.Errorf("get appconfig by name failed: %s, except name: %s type: %s, actual name: %s type: %s", err, name, _type, appconfig.Name, appconfig.Type)
	}
}

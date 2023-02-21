package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

func getChainDataByAuthType(urls []string, payload AuthTypeItem, ciperKey []byte) (*handlers.TrxAuthItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: AUTH_TYPE,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result handlers.TrxAuthItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func getChainDataByAuthAllowList(urls []string, payload AuthAllowListItem, ciperKey []byte) ([]handlers.ChainSendTrxRuleListItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: AUTH_ALLOWLIST,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result []handlers.ChainSendTrxRuleListItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, true); err != nil {
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

func getChainDataByAuthDenyList(urls []string, payload AuthDenyListItem, ciperKey []byte) ([]*handlers.ChainSendTrxRuleListItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: AUTH_DENYLIST,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result []*handlers.ChainSendTrxRuleListItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, true); err != nil {
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

func getChainDataByAppConfigKeyList(urls []string, payload AppConfigKeyListItem, ciperKey []byte) ([]*handlers.AppConfigKeyListItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: APPCONFIG_KEYLIST,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result []*handlers.AppConfigKeyListItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, true); err != nil {
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

func getChainDataByAppConfigItemByKey(urls []string, payload AppConfigItem, ciperKey []byte) (*handlers.AppConfigKeyItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: APPCONFIG_ITEM_BYKEY,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result handlers.AppConfigKeyItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func getChainDataByAnnouncedProducer(urls []string, payload AnnGrpProducer, ciperKey []byte) ([]*handlers.AnnouncedProducerListItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: ANNOUNCED_PRODUCER,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result []*handlers.AnnouncedProducerListItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, true); err != nil {
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

func getChainDataByAnnouncedUser(urls []string, payload AnnGrpUser, ciperKey []byte) ([]*handlers.AnnouncedUserListItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: ANNOUNCED_USER,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result []*handlers.AnnouncedUserListItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, true); err != nil {
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

func getChainDataByGroupProducer(urls []string, payload GrpProducer, ciperKey []byte) ([]*handlers.ProducerListItem, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: GROUP_PRODUCER,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result []*handlers.ProducerListItem
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, true); err != nil {
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

func getChainDataByGroupInfo(urls []string, payload GrpInfo, ciperKey []byte) (*GrpInfoNodeSDK, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	encrypted, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	param := GetDataNodeSDKItem{
		GroupId: payload.GroupId,
		ReqType: GROUP_INFO,
		Req:     encrypted,
	}

	path := fmt.Sprintf("/api/v1/node/getchaindata/%s", payload.GroupId)
	var result GrpInfoNodeSDK
	if _, _, err := requestNSdk(urls, path, "POST", param, nil, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestGetChainDataNSdk(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-chain-data",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	seed, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}

	ciperKey, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		t.Errorf("convert seed.CipherKey failed: %s", err)
	}

	if _, err := getChainDataByGroupInfo(urls, GrpInfo{GroupId: group.GroupId}, ciperKey); err != nil {
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
	time.Sleep(5 * time.Second)
	authTypeResult, err := getChainDataByAuthType(urls, AuthTypeItem{GroupId: group.GroupId, TrxType: _trxType}, ciperKey)
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
	time.Sleep(5 * time.Second)

	if _, err := getChainDataByAuthAllowList(urls, AuthAllowListItem{GroupId: group.GroupId}, ciperKey); err != nil {
		t.Errorf("getChainDataByAuthAllowList failed: %s", err)
	}
}

func TestGetChainDataAppConfigNSdk(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-chain-data",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	seed, urls, err := handlers.UrlToGroupSeed(group.Seed)
	if err != nil {
		t.Errorf("convert group send url failed: %s", err)
	}

	ciperKey, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		t.Errorf("convert seed.CipherKey failed: %s", err)
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

	time.Sleep(3 * time.Second)

	keylist, err := getChainDataByAppConfigKeyList(urls, AppConfigKeyListItem{GroupId: group.GroupId}, ciperKey)
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
		urls, AppConfigItem{GroupId: group.GroupId, Key: name}, ciperKey,
	)
	if err != nil {
		t.Errorf("get appconfig by name: %s failed: %s", err, name)
	}

	if appconfig.Name != name || appconfig.Type != strings.ToUpper(_type) {
		t.Errorf("get appconfig by name failed: %s, except name: %s type: %s, actual name: %s type: %s", err, name, _type, appconfig.Name, appconfig.Type)
	}
}

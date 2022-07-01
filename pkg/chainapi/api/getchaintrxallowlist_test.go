package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/go-cmp/cmp"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func TestUpdateChainAllowList(t *testing.T) {
	// create group
	payload := handlers.CreateGroupParam{
		AppKey:         "default",
		ConsensusType:  "poa",
		EncryptionType: "public",
		GroupName:      fmt.Sprintf("public-%d", time.Now().Unix()),
	}

	group, err := createGroup(peerapi, payload)
	if err != nil {
		t.Fatalf("create group failed: %s", err)
	}

	trxAuthModes := []string{"follow_alw_list", "follow_dny_list"}
	for _, trxAuthMode := range trxAuthModes {
		for _, trxType := range trxTypes {
			authParams := handlers.TrxAuthModeParams{
				TrxType:     trxType,
				TrxAuthMode: trxAuthMode,
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
		}
	}

	// update allow list
	config := handlers.ChainSendTrxRuleListItemParams{
		Action:  "add",
		Pubkey:  "CAISIQJEqFVXRAzzfE5V9/xHyzIBE464qzdzY1OaeDzI4Ihr01==",
		TrxType: []string{trxTypes[rand.Int()%len(trxTypes)]},
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal config failed: %s", err)
	}
	chainConfigParams := handlers.ChainConfigParams{
		GroupId: group.GroupId,
		Type:    "upd_alw_list",
		Config:  string(configBytes),
	}
	if _, err := updateChainConfig(peerapi, chainConfigParams); err != nil {
		t.Fatalf("updateChainConfig with config: %+v failed: %s", chainConfigParams, err)
	}

	// get allow list
	time.Sleep(10 * time.Second)
	rules, err := getGroupAllowList(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getGroupAllowList failed: %s", err)
	}
	if rules == nil || len(rules) == 0 {
		t.Fatalf("group chain config should not be empty")
	}

	found := false
	for _, rule := range rules {
		if rule.Pubkey == config.Pubkey && cmp.Equal(rule.TrxType, config.TrxType) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("should get chain config rule from allow list")
	}
}

func getGroupAllowList(api string, groupId string) ([]handlers.ChainSendTrxRuleListItem, error) {
	url := fmt.Sprintf("/api/v1/group/%s/trx/allowlist", groupId)
	_, resp, err := testnode.RequestAPI(api, url, "GET", "")
	if err != nil {
		e := fmt.Errorf("get group allow list for %s failed: %s", groupId, err)
		return nil, e
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var allowList []handlers.ChainSendTrxRuleListItem
	if err := json.Unmarshal(resp, &allowList); err != nil {
		e := fmt.Errorf("response Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	for _, item := range allowList {
		validate := validator.New()
		if err := validate.Struct(item); err != nil {
			return nil, err
		}
	}

	return allowList, nil
}

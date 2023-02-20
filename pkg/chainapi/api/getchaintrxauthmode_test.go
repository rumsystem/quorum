package api

import (
	"fmt"
	"testing"
	"time"

	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

var (
	trxTypes = []string{
		"POST", "ANNOUNCE", "REQ_BLOCK",

		// NOTE: this trx type can not be configured
		// "PRODUCER",  "USER", "CHAIN_CONFIG", "APP_CONFIG",
	}
)

func getChainTrxAuthMode(api string, payload handlers.TrxAuthParams) (*handlers.TrxAuthItem, error) {
	path := fmt.Sprintf("/api/v1/group/%s/trx/auth/%s", payload.GroupId, payload.TrxType)
	var authItem handlers.TrxAuthItem
	_, _, err := requestAPI(api, path, "GET", nil, nil, &authItem, false)
	if err != nil {
		e := fmt.Errorf("get chain trx auth mode for (%s, %s) failed: %s", payload.GroupId, payload.TrxType, err)
		return nil, e
	}

	if authItem.TrxType != payload.TrxType {
		return nil, fmt.Errorf("trx type not match, expect: %s, actual: %s", payload.TrxType, authItem.TrxType)
	}
	if authItem.AuthType != "FOLLOW_DNY_LIST" && authItem.AuthType != "FOLLOW_ALW_LIST" {
		return nil, fmt.Errorf("auth type not match, expect: FOLLOW_DNY_LIST or FOLLOW_ALW_LIST, actual: %s", authItem.AuthType)
	}

	return &authItem, nil
}

func TestGetChainTrxAuthModeForNewGroup(t *testing.T) {
	// create group
	payload := handlers.CreateGroupParam{
		AppKey:         "default",
		ConsensusType:  "poa",
		EncryptionType: "public",
		GroupName:      fmt.Sprintf("public-%d", time.Now().Unix()),
	}

	group, err := createGroup(peerapi, payload)
	if err != nil {
		t.Errorf("create group failed: %s", err)
	}

	for _, trxType := range trxTypes {
		params := handlers.TrxAuthParams{
			GroupId: group.GroupId,
			TrxType: trxType,
		}
		authItem, err := getChainTrxAuthMode(peerapi, params)
		if err != nil {
			t.Errorf("get chain trx auth mode failed: %s", err)
		}

		if authItem.AuthType != "FOLLOW_DNY_LIST" {
			t.Errorf("auth type not match, expect: %s, actual: %s", "FOLLOW_DNY_LIST", authItem.AuthType)
		}
	}

	allowList, err := getGroupAllowList(peerapi, group.GroupId)
	if err != nil {
		t.Errorf("get group allow list for %s failed: %s", group.GroupId, err)
	}
	if allowList != nil {
		t.Errorf("group allow list for new group should be null")
	}

	denyList, err := getGroupDenyList(peerapi, group.GroupId)
	if err != nil {
		t.Errorf("get group deny list for %s failed: %s", group.GroupId, err)
	}
	if denyList != nil {
		t.Errorf("group deny list for new group should be null")
	}
}

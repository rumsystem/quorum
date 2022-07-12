package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

var (
	trxTypes = []string{
		"POST", "ANNOUNCE", "REQ_BLOCK_FORWARD", "REQ_BLOCK_BACKWARD", "BLOCK_SYNCED",
		"BLOCK_PRODUCED", "ASK_PEERID",
	}
)

func getChainTrxAuthMode(api string, payload handlers.TrxAuthParams) (*handlers.TrxAuthItem, error) {
	url := fmt.Sprintf("/api/v1/group/%s/trx/auth/%s", payload.GroupId, payload.TrxType)
	_, resp, err := testnode.RequestAPI(api, url, "GET", "")
	if err != nil {
		e := fmt.Errorf("get chain trx auth mode for (%s, %s) failed: %s", payload.GroupId, payload.TrxType, err)
		return nil, e
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var authItem handlers.TrxAuthItem
	if err := json.Unmarshal(resp, &authItem); err != nil {
		e := fmt.Errorf("response Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	validate := validator.New()
	if err := validate.Struct(authItem); err != nil {
		return nil, err
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

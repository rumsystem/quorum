package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func TestSetChainTrxAuthMode(t *testing.T) {
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

			// wait 10 seconds
			time.Sleep(15 * time.Second)
			authItem, err := getChainTrxAuthMode(peerapi, handlers.TrxAuthParams{GroupId: group.GroupId, TrxType: trxType})
			if err != nil {
				t.Errorf("get chain trx auth mode failed: %s", err)
			}
			if authItem.AuthType != strings.ToUpper(trxAuthMode) {
				t.Errorf("auth type not match, expect: %s, actual: %s", strings.ToUpper(trxAuthMode), authItem.AuthType)
			}
		}
	}
}
func updateChainConfig(api string, payload handlers.ChainConfigParams) (*handlers.ChainConfigResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		e := fmt.Errorf("json.Marshal failed, payload: %+v error: %s", payload, err)
		return nil, e
	}
	payloadStr := string(payloadByte[:])

	_, resp, err := testnode.RequestAPI(api, "/api/v1/group/chainconfig", "POST", payloadStr)
	if err != nil {
		e := fmt.Errorf("update group chain config failed: %s", err)
		return nil, e
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.ChainConfigResult
	if err := json.Unmarshal(resp, &result); err != nil {
		e := fmt.Errorf("response Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

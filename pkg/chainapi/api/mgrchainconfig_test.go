package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func TestSetChainTrxAuthMode(t *testing.T) {
	t.Parallel()

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

			time.Sleep(5 * time.Second)
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
	var result handlers.ChainConfigResult
	_, _, err := requestAPI(api, "/api/v1/group/chainconfig", "POST", payload, &result)
	if err != nil {
		e := fmt.Errorf("update group chain config failed: %s", err)
		return nil, e
	}

	return &result, nil
}

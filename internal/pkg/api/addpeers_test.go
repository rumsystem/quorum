package api

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/testnode"
)

func addPeers(api string, payload AddPeerParam) (*AddPeerResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadStr := string(payloadByte)

	urlSuffix := "/api/v1/network/peers"
	resp, err := testnode.RequestAPI(api, urlSuffix, "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result AddPeerResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestAddPeers(t *testing.T) {
	payload := AddPeerParam{"/ip4/94.23.17.189/tcp/10666/p2p/16Uiu2HAkvYJbiKnwzFybbfzNxtuHFEFu1guV9nFcTiZNSYz8kUWf"}

	if _, err := addPeers(peerapi, payload); err != nil {
		t.Errorf("addPeers failed: %s, payload: %+v", err, payload)
	}
}

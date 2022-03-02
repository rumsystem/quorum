package api

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func TestGetNetwork(t *testing.T) {
	_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/network", "GET", "")
	if err != nil {
		t.Errorf("get network failed: %s", err)
	}

	if err := getResponseError(resp); err != nil {
		t.Errorf("request failed: %s, response: %s", err, resp)
	}

	var network handlers.NetworkInfo
	if err := json.Unmarshal(resp, &network); err != nil {
		t.Errorf("response data Unmarshal error: %s", err)
	}

	validate := validator.New()
	if err := validate.Struct(network); err != nil {
		t.Errorf("response data invalid: %s, response: %+v", err, network)
	}
}

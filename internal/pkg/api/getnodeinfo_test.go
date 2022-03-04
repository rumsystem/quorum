package api

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func getNodeInfo(api string) (*handlers.NodeInfo, error) {
	_, resp, err := testnode.RequestAPI(api, "/api/v1/node", "GET", "")
	if err != nil {
		return nil, err
	}

	var info handlers.NodeInfo
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err != validate.Struct(info) {
		return nil, err
	}

	return &info, nil
}

func getNodePublicKey(api string) (string, error) {
	info, err := getNodeInfo(api)
	if err != nil {
		return "", err
	}

	return info.NodePublickey, nil
}

func TestGetNodeInfo(t *testing.T) {
	if _, err := getNodeInfo(peerapi); err != nil {
		t.Fatalf("getNodeInfo failed: %s", err)
	}
}

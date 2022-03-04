package api

import (
	"encoding/json"
	"testing"

	"github.com/rumsystem/quorum/testnode"
)

func TestGetBootstrapNodeInfo(t *testing.T) {
	urlSuffix := "/api/v1/node"
	_, resp, err := testnode.RequestAPI(bootstrapapi, urlSuffix, "GET", "")
	if err != nil {
		t.Errorf("GET %s failed: %s", urlSuffix, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		t.Errorf("json.Unmarshal failed: %s, response: %s", err, resp)
	}

	expectKeys := []string{"node_status", "node_type", "node_id"}
	keys := GetMapKeys(data)
	if !StringSetEqual(expectKeys, keys) {
		t.Errorf("unexpected keys: %v, expect: %v", keys, expectKeys)
	}
	if data["node_status"] != "NODE_ONLINE" {
		t.Errorf("unexpected node_status: %s", data["node_status"])
	}
	if data["node_type"] != "bootstrap" {
		t.Errorf("unexpected node_type: %s", data["node_type"])
	}
}

package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/testnode"
)

func backup(api string) (*BackupResult, error) {
	_, resp, err := testnode.RequestAPI(api, "/api/v1/backup", "GET", "")
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result BackupResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %s response: %+v", err, resp)
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, fmt.Errorf("validate.Struct failed: %s", err)
	}

	return &result, nil
}

func TestBackup(t *testing.T) {
	if _, err := backup(peerapi); err != nil {
		t.Fatal(err)
	}
}

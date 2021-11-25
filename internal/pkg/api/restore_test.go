package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/testnode"
)

func restore(api string, payload RestoreParam) (*RestoreResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %s", err)
	}

	payloadStr := string(payloadBytes)
	resp, err := testnode.RequestAPI(api, "/api/v1/restore", "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result RestoreResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %s response: %+v", err, resp)
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, fmt.Errorf("validate.Struct failed: %s", err)
	}

	if !utils.DirExist(result.Path) {
		return nil, fmt.Errorf("restore path %s not exist", result.Path)
	}

	return &result, nil
}

func TestRestore(t *testing.T) {
	// create group
	createGroupParam := CreateGroupParam{
		GroupName:      "test-join-group",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("create group failed: %s, payload: %+v", err, createGroupParam)
	}

	// backup
	backupResult, err := backup(peerapi)
	if err != nil {
		t.Fatalf("backup failed: %s", err)
	}

	// restore
	path := t.TempDir()
	restoreParam := RestoreParam{
		BackupResult: *backupResult,
		Password:     testnode.KeystorePassword,
		Path:         path,
	}

	result, err := restore(peerapi, restoreParam)
	if err != nil {
		t.Fatal(err)
	}

	seedPath := filepath.Join(result.Path, "seeds.json")
	if !utils.FileExist(seedPath) {
		t.Errorf("seeds.json not exist")
	}

	// load seeds.json
	seedBytes, err := ioutil.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("ioutil.ReadFile failed: %s", err)
	}

	var seeds []GroupSeed
	if err := json.Unmarshal(seedBytes, &seeds); err != nil {
		t.Fatalf("json.Unmarshal failed: %s", err)
	}

	found := false
	for _, seed := range seeds {
		if seed.GroupId != group.GroupId {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("group %s not found in seeds.json", group.GroupId)
	}
}

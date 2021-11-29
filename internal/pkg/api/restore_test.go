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

	seedPath := filepath.Join(result.Path, "seeds")
	if !utils.DirExist(seedPath) {
		t.Errorf("seeds directory not exist")
	}

	empty, err := utils.IsDirEmpty(seedPath)
	if err != nil {
		t.Errorf("check seeds directory empty failed: %s", err)
	}
	if empty {
		t.Errorf("seeds directory is empty")
	}

	// load seed json
	seedfiles, err := ioutil.ReadDir(seedPath)
	if err != nil {
		t.Errorf("read seeds directory failed: %s", err)
	}

	var seeds []GroupSeed
	for _, seedfile := range seedfiles {
		path := filepath.Join(seedPath, seedfile.Name())
		seedBytes, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatalf("ioutil.ReadFile failed: %s", err)
		}

		var seed GroupSeed
		if err := json.Unmarshal(seedBytes, &seed); err != nil {
			t.Fatalf("json.Unmarshal failed: %s", err)
		}

		seeds = append(seeds, seed)
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

package api

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/testnode"
)

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
	backupFile := filepath.Join(t.TempDir(), "backup.json")
	backupContent, err := json.Marshal(backupResult)
	if err != nil {
		t.Fatalf("json.Marshal failed: %s", err)
	}
	if err := ioutil.WriteFile(backupFile, backupContent, 0644); err != nil {
		t.Fatalf("ioutil.WriteFile failed: %s", err)
	}

	// restore
	path := t.TempDir()
	restoreParam := RestoreParam{
		BackupFile:  backupFile,
		Password:    testnode.KeystorePassword,
		ConfigDir:   filepath.Join(path, "config"),
		KeystoreDir: filepath.Join(path, "keystore"),
		SeedDir:     filepath.Join(path, "seeds"),
	}

	if err := Restore(restoreParam); err != nil {
		t.Fatalf("Restore failed: %s", err)
	}

	// check config directory
	if !utils.DirExist(restoreParam.ConfigDir) {
		t.Errorf("config directory not exist")
	}
	empty, err := utils.IsDirEmpty(restoreParam.ConfigDir)
	if err != nil {
		t.Fatalf("utils.IsDirEmpty failed: %s", err)
	}
	if empty {
		t.Errorf("config directory is empty")
	}

	// check keystore directory
	if !utils.DirExist(restoreParam.KeystoreDir) {
		t.Errorf("keystore directory not exist")
	}
	empty, err = utils.IsDirEmpty(restoreParam.KeystoreDir)
	if err != nil {
		t.Fatalf("utils.IsDirEmpty failed: %s", err)
	}
	if empty {
		t.Errorf("keystore directory is empty")
	}

	// check group seed
	seedPath := restoreParam.SeedDir
	if !utils.DirExist(seedPath) {
		t.Errorf("seeds directory not exist")
	}

	empty, err = utils.IsDirEmpty(seedPath)
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

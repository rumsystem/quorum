package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type BackupResult struct {
	// encrypt json.Marshal([]GroupSeed)
	Seeds    string `json:"seeds"`
	Keystore string `json:"keystore"`
	Config   string `json:"config" validate:"required"`
}

// @Tags Chain
// @Summary Backup
// @Description Backup my group seed/keystore/config
// @Produce json
// @Success 200 {object} BackupResult
// @Router /api/v1/group/backup [get]
func (h *Handler) Backup(c echo.Context) (err error) {
	// get all the seed of joined groups
	seeds, err := getGroupSeeds(h.Appdb)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("get group seeds failed: %s", err)})
	}

	seedsBytes, err := zipGroupSeeds(seeds)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("zipGroupSeeds failed: %s", err)})
	}

	ks := nodectx.GetNodeCtx().Keystore
	encSeedsStr, encKeystoreStr, encConfigStr, err := ks.Backup(seedsBytes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("backup failed: %s", err)})
	}

	result := BackupResult{
		Keystore: encKeystoreStr,
		Config:   encConfigStr,
		Seeds:    encSeedsStr,
	}

	return c.JSON(http.StatusOK, result)
}

// get myself group seeds
func getGroupSeeds(appdb *appdata.AppDb) ([]GroupSeed, error) {
	var seeds []GroupSeed
	groupmgr := chain.GetGroupMgr()

	for _, item := range groupmgr.Groups {
		groupID := item.Item.GroupId
		pbSeed, err := appdb.GetGroupSeed(groupID)
		if err != nil {
			return nil, fmt.Errorf("appdb.GetGroupSeed failed: %s", err)
		}
		if pbSeed == nil {
			return nil, fmt.Errorf("group seed not found: %s", groupID)
		}

		seed := FromPbGroupSeed(pbSeed)
		seeds = append(seeds, seed)
	}

	return seeds, nil
}

// zip group seeds
func zipGroupSeeds(seeds []GroupSeed) ([]byte, error) {
	// cannot write to the system's temporary directory in the mobile application
	// so use the data directory
	dataDir := cli.GetConfig().DataDir
	if len(dataDir) == 0 {
		return nil, fmt.Errorf("can not get data directory")
	}

	tempDir, err := ioutil.TempDir(dataDir, "")
	if err != nil {
		return nil, fmt.Errorf("create temp dir failed: %s", err)
	}

	defer os.RemoveAll(tempDir)

	seedDir := filepath.Join(tempDir, "seeds")
	if err := utils.EnsureDir(seedDir); err != nil {
		return nil, fmt.Errorf("utils.EnsureDir failed: %s", err)
	}

	for _, seed := range seeds {
		seedByte, err := json.MarshalIndent(seed, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal group seed failed: %s", err)
		}

		path := filepath.Join(seedDir, fmt.Sprintf("%s.json", seed.GroupId))
		if err := ioutil.WriteFile(path, seedByte, 0644); err != nil {
			return nil, fmt.Errorf("write group seed failed: %s", err)
		}
	}

	data, err := utils.ZipDir(seedDir)
	if err != nil {
		return nil, fmt.Errorf("zip group seeds failed: %s", err)
	}

	return data, nil
}

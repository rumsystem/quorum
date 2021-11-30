package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
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

	seedsBytes, err := json.MarshalIndent(seeds, "", "  ")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("marshal group seeds failed: %s", err)})
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
			if errors.Is(err, appdata.ErrNotFound) {
				continue
			}

			return nil, fmt.Errorf("appdb.GetGroupSeed failed: %s", err)
		}

		seed := FromPbGroupSeed(pbSeed)
		seeds = append(seeds, seed)
	}

	return seeds, nil
}

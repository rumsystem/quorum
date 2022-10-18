package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

// get myself group seeds
func GetAllGroupSeeds(appdb *appdata.AppDb) ([]CreateGroupResult, error) {

	pbSeeds, err := appdb.GetAllGroupSeeds()
	if err != nil {
		return nil, err
	}

	var seeds []CreateGroupResult
	for _, value := range pbSeeds {
		seed := FromPbGroupSeed(value)

		// FIXME: hardcode version 1
		seedUrl, err := GroupSeedToUrl(1, nil, &seed)
		if err != nil {
			logger.Errorf("GroupSeedToUrl failed: %s", err)
			return nil, err
		}

		item := CreateGroupResult{
			Seed:    seedUrl,
			GroupId: seed.GroupId,
		}
		seeds = append(seeds, item)
	}

	return seeds, nil
}

// saveAllGroupSeeds save group seed to `seedDir`
func SaveAllGroupSeeds(appdb *appdata.AppDb, seedDir string) error {
	if err := utils.EnsureDir(seedDir); err != nil {
		return err
	}

	seeds, err := GetAllGroupSeeds(appdb)
	if err != nil {
		return err
	}

	for _, seed := range seeds {
		seedByte, err := json.MarshalIndent(seed, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal group seed failed: %s", err)
		}

		path := filepath.Join(seedDir, fmt.Sprintf("%s.json", seed.GroupId))
		if err := ioutil.WriteFile(path, seedByte, 0644); err != nil {
			return fmt.Errorf("write group seed failed: %s", err)
		}
	}

	return nil
}

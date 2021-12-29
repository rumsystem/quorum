package handlers

import (
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
)

func GetGroupSeed(groupId string, appdb *appdata.AppDb) (*GroupSeed, error) {
	pbSeed, err := appdb.GetGroupSeed(groupId)
	if err != nil {
		return nil, fmt.Errorf("get group seeds failed: %s", err)
	}

	if pbSeed == nil {
		return nil, errors.New("group seed not found")
	}

	seed := FromPbGroupSeed(pbSeed)

	return &seed, nil
}

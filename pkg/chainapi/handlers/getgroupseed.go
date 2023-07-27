package handlers

import (
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/pkg/pb"
)

type GetGroupSeedParam struct {
	GroupId         string `param:"group_id" validate:"required,uuid4" example:"19fbf6d8-90d1-450e-82b0-eaf9e38bc55b"`
	IncludeChainUrl bool   `query:"include_chain_url" example:"true"`
}

func GetGroupSeed(groupId string, appdb *appdata.AppDb) (*pb.GroupSeed, error) {
	pbSeed, err := appdb.GetGroupSeed(groupId)
	if err != nil {
		return nil, fmt.Errorf("get group seeds failed: %s", err)
	}

	if pbSeed == nil {
		return nil, errors.New("group seed not found")
	}

	return pbSeed, nil
}

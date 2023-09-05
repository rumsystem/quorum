package handlers

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GetGroupSeedParam struct {
	GroupId string `param:"group_id" validate:"required,uuid4" example:"19fbf6d8-90d1-450e-82b0-eaf9e38bc55b"`
	//IncludeChainUrl bool   `query:"include_chain_url" example:"true"`
}

type GetGroupSeedResult struct {
	Seed     *quorumpb.GroupSeed `json:"seed"` //
	SeedByts []byte              `json:"seed_byts"`
}

func GetGroupSeed(groupId string, appdb *appdata.AppDb) (*quorumpb.GroupSeed, error) {
	seed, err := appdb.GetGroupSeed(groupId)
	if err != nil {
		return nil, fmt.Errorf("get group seeds failed: %s", err)
	}

	return seed, nil
}

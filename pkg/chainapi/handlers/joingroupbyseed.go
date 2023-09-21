package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	"github.com/rumsystem/quorum/pkg/pb"
	_ "github.com/rumsystem/quorum/pkg/pb" //import for swaggo
	"google.golang.org/protobuf/proto"
)

type JoinGroupBySeedParams struct {
	Seed            []byte `from:"seed" json:"seed" validate:"required"`
	OwnerKeyname    string `from:"user_keyname" json:"owner_keyname"`
	PosterKeyname   string `from:"user_keyname" json:"poster_keyname"`
	SyncerKeyname   string `from:"user_keyname" json:"syncer_keyname"`
	ProducerKeyname string `from:"user_keyname" json:"producer_keyname"`
}

type JoinGroupBySeedResult struct {
	GroupId string `json:"group_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
}

func JoinGroupBySeed(params *JoinGroupBySeedParams, nodeoptions *options.NodeOptions) (*JoinGroupBySeedResult, error) {
	//unmarshal seed
	seed := &pb.GroupSeed{}
	err := proto.Unmarshal(params.Seed, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//check if group alreay exist
	isExist, err := chain.GetGroupMgr().IsParentGroupExist(chaindef.JOIN_BY_API)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}
	if isExist {
		msg := fmt.Sprintf("group with group_id <%s> already exist", seed.GroupId)
		return nil, rumerrors.NewBadRequestError(msg)
	}

	verified, err := rumchaindata.VerifyGroupSeed(seed)

	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	if !verified {
		return nil, rumerrors.NewBadRequestError("seed not verified")
	}

	//create empty group
	group := &chain.Group{}
	err = group.JoinGroupBySeed(chaindef.JOIN_BY_API,
		params.OwnerKeyname,
		params.PosterKeyname,
		params.ProducerKeyname,
		params.SyncerKeyname, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//add group to groupMgr
	chain.GetGroupMgr().AddSubGroup(chaindef.JOIN_BY_API, group.GroupItem)

	//save group seed
	if err := nodectx.GetNodeCtx().GetChainStorage().SetGroupSeed(seed); err != nil {
		msg := fmt.Sprintf("save group seed failed: %s", err)
		return nil, rumerrors.NewBadRequestError(msg)
	}

	return &JoinGroupBySeedResult{GroupId: group.GroupId}, nil
}

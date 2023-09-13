package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	"github.com/rumsystem/quorum/pkg/pb"
	_ "github.com/rumsystem/quorum/pkg/pb" //import for swaggo
	"google.golang.org/protobuf/proto"
)

type JoinGroupBySeedParams struct {
	Seed        []byte `from:"seed" json:"seed" validate:"required"`
	UserKeyName string `from:"user_keyname" json:"user_keyname" validate:"required"`
}

type JoinGroupBySeedResult struct {
	GroupId string `json:"group_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
}

func JoinGroupBySeed(params *JoinGroupBySeedParams, nodeoptions *options.NodeOptions) (*JoinGroupBySeedResult, error) {
	ks := localcrypto.GetKeystore()

	//check if trx sign keyname exist
	userSignPubkey, err := ks.GetEncodedPubkey(params.UserKeyName, localcrypto.Sign)
	if err != nil {
		return nil, errors.New("trx sign keyname not found in local keystore")
	}

	//unmarshal seed
	seed := &pb.GroupSeed{}
	err = proto.Unmarshal(params.Seed, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//check if group alreay exist
	groupmgr := chain.GetGroupMgr()
	if _, ok := groupmgr.Groups[seed.GroupId]; ok {
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
	err = group.JoinGroupBySeed(userSignPubkey, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//add group to context
	groupmgr.Groups[group.Item.GroupId] = group

	//save group seed
	if err := nodectx.GetNodeCtx().GetChainStorage().SetGroupSeed(seed); err != nil {
		msg := fmt.Sprintf("save group seed failed: %s", err)
		return nil, rumerrors.NewBadRequestError(msg)
	}

	return &JoinGroupBySeedResult{GroupId: group.GroupId}, nil
}

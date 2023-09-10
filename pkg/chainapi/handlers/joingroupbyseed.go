package handlers

import (
	"bytes"
	"encoding/hex"
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
	Seed            []byte `from:"seed" json:"seed" validate:"required"`
	UserSignKeyName string `from:"user_sign_keyname" json:"user_sign_keyname" validate:"required"`
}

type JoinGroupBySeedResult struct {
	GroupItem *pb.GroupItem `json:"groupItem"`
}

func JoinGroupBySeed(params *JoinGroupBySeedParams, nodeoptions *options.NodeOptions) (*JoinGroupBySeedResult, error) {
	ks := localcrypto.GetKeystore()

	//check if trx sign keyname exist
	userSignPubkey, err := ks.GetEncodedPubkey(params.UserSignKeyName, localcrypto.Sign)
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

	seedClone := proto.Clone(seed).(*pb.GroupSeed)
	seedClone.Hash = nil
	seedClone.Signature = nil

	//verify hash and signature
	seedCloneByts, err := proto.Marshal(seedClone)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}
	hash := localcrypto.Hash(seedCloneByts)
	if !bytes.Equal(hash, seed.Hash) {
		msg := fmt.Sprintf("hash not match, expect %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(seed.Hash))
		return nil, rumerrors.NewBadRequestError(msg)
	}

	verified, err := rumchaindata.VerifySign(seed.OwnerPubkey, hash, seed.Signature)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	if !verified {
		return nil, rumerrors.NewBadRequestError("verify signature failed")
	}

	//verify genesis block
	r, err := rumchaindata.ValidGenesisBlockPoa(seed.GenesisBlock)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	if !r {
		msg := "Join Group failed, verify genesis block failed"
		return nil, rumerrors.NewBadRequestError(msg)
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

	return &JoinGroupBySeedResult{GroupItem: group.Item}, nil
}

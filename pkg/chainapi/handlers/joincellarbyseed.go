package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
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

type JoinCellarBySeedParams struct {
	Seed            []byte `from:"seed" json:"seed" validate:"required"`
	UserSignKeyName string `from:"user_sign_keyname" json:"user_sign_keyname" validate:"required"`
}

type JoinCellarBySeedResult struct {
	CellarId string `json:"cellar_id" validate:"required"`
}

func JoinCellarBySeed(params *JoinCellarBySeedParams, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*JoinCellarBySeedResult, error) {
	ks := localcrypto.GetKeystore()

	//check if trx sign keyname exist
	userSignPubkey, err := ks.GetEncodedPubkey(params.UserSignKeyName, localcrypto.Sign)
	if err != nil {
		return nil, errors.New("trx sign keyname not found in local keystore")
	}

	//unmarshal seed
	seed := &pb.CellarSeed{}
	err = proto.Unmarshal(params.Seed, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//check if cellar alreay exist
	isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsCellarExist(seed.CellarId)
	if err != nil {
		return nil, err
	}

	if isExist {
		msg := fmt.Sprintf("cellar with cellar_id <%s> already exist", seed.CellarId)
		return nil, rumerrors.NewBadRequestError(msg)
	}

	seedClone := proto.Clone(seed).(*pb.CellarSeed)
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

	verified, err := rumchaindata.VerifySign(seed.CellarOwnerPubkey, hash, seed.Signature)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	if !verified {
		return nil, rumerrors.NewBadRequestError("verify signature failed")
	}

	//TBD verify cellar group seed
	groupSeedClone := proto.Clone(seed.CellarGroupSeed).(*pb.GroupSeed)
	groupSeedClone.Hash = nil
	groupSeedClone.Signature = nil
	groupSeedCloneByts, err := proto.Marshal(groupSeedClone)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	hash = localcrypto.Hash(groupSeedCloneByts)
	if !bytes.Equal(hash, seed.CellarGroupSeed.Hash) {
		msg := fmt.Sprintf("hash not match, expect %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(seed.CellarGroupSeed.Hash))
		return nil, rumerrors.NewBadRequestError(msg)
	}

	verified, err = rumchaindata.VerifySign(seed.CellarGroupSeed.OwnerPubkey, hash, seed.CellarGroupSeed.Signature)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//create empty group
	cellar := &chain.Cellar{}
	err = cellar.JoinCellarBySeed(userSignPubkey, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	//add group to context
	//cellarMgr.Groups[group.Item.GroupId] = group

	//save group seed
	//if err := appdb.SetGroupSeed(seed); err != nil {
	//	msg := fmt.Sprintf("save group seed failed: %s", err)
	//	return nil, rumerrors.NewBadRequestError(msg)
	//}

	return &JoinCellarBySeedResult{CellarId: seed.CellarId}, nil
}

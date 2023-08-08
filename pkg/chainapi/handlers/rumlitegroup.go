package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	_ "github.com/rumsystem/quorum/pkg/pb" //import for swaggo
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type JoinGroupBySeedParams struct {
	Seed           []byte `from:"seed" json:"seed" validate:"required"`
	TrxSignKeyName string `from:"trx_sign_keyname" json:"trx_sign_keyname" validate:"required"`
}
type JoinGroupBySeedResult struct {
	GroupItem *quorumpb.GroupItemRumLite `json:"groupItem"`
}

func JoinGroupBySeed(params *JoinGroupBySeedParams, nodeoptions *options.NodeOptions) (*JoinGroupBySeedResult, error) {
	ks := localcrypto.GetKeystore()

	//check if trx sign keyname exist
	trxSignPubkey, err := ks.GetEncodedPubkey(params.TrxSignKeyName, localcrypto.Sign)
	if err != nil {
		return nil, errors.New("trx sign keyname not found in local keystore")
	}

	//unmarshal seed
	seed := &quorumpb.GroupSeedRumLite{}
	err = proto.Unmarshal(params.Seed, seed)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	groupItem := seed.Group

	//check if group alreay exist
	groupmgr := chain.GetGroupMgr()
	if _, ok := groupmgr.Groups[groupItem.GroupId]; ok {
		msg := fmt.Sprintf("group with group_id <%s> already exist", groupItem.GroupId)
		return nil, rumerrors.NewBadRequestError(msg)
	}

	//verify hash and signature
	groupItemByts, err := proto.Marshal(groupItem)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}
	hash := localcrypto.Hash(groupItemByts)
	if !bytes.Equal(hash, seed.Hash) {
		msg := fmt.Sprintf("hash not match, expect %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(seed.Hash))
		return nil, rumerrors.NewBadRequestError(msg)
	}

	verified, err := rumchaindata.VerifySign(groupItem.OwnerPubKey, hash, seed.Signature)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	if !verified {
		return nil, rumerrors.NewBadRequestError("verify signature failed")
	}

	//verify genesis block
	r, err := rumchaindata.ValidGenesisBlockRumLite(groupItem.GenesisBlock)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	if !r {
		msg := "Join Group failed, verify genesis block failed"
		return nil, rumerrors.NewBadRequestError(msg)
	}

	//create empty group
	group := &chain.GroupRumLite{}
	groupItem.TrxSignPubkey = trxSignPubkey
	err = group.JoinGroup(groupItem)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(err)
	}

	return &JoinGroupBySeedResult{GroupItem: groupItem}, nil
}

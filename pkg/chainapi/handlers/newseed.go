package handlers

import (
	"encoding/hex"
	"errors"

	"github.com/go-playground/validator/v10"
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	"github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type NewSeedParams struct {
	AppKey         string `from:"app_key"         json:"app_key"         validate:"required,max=20,min=4" example:"test_app"`
	GroupName      string `from:"group_name"      json:"group_name"      validate:"required,max=100,min=2" example:"demo group"`
	ConsensusType  string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa" example:"poa"`
	EncryptionType string `from:"encryption_type" json:"encryption_type" validate:"required,oneof=public private" example:"public"`
}

type NewSeedResult struct {
	AppKey  string               `json:"app_key"           validate:"required" example:"test_app"`
	GroupId string               `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	Seed    *pb.GroupSeedRumLite `json:"seed" validate:"required"`
}

const DEFAULT_EPOCH_DURATION = 1000 //ms

func NewSeed(params *NewSeedParams, nodeoptions *options.NodeOptions) (*NewSeedResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	if params.ConsensusType != "poa" {
		return nil, errors.New("consensus_type must be poa, other types are not supported in rum-lite")
	}

	if params.EncryptionType != "public" {
		return nil, errors.New("encryption_type must be public, other types are not supported in rum-lite")
	}

	//create groupid
	groupid := guuid.New().String()

	//init keystore
	ks := nodectx.GetNodeCtx().Keystore

	//init ownerpubkey
	ownerpubkey, err := initSignKey(groupid, ks, nodeoptions)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	//init cipher key
	cipherKey, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}

	//create genesis block
	genesisBlock, err := rumchaindata.CreateGenesisBlockByEthKey(groupid, ownerpubkey, ks, "")
	if err != nil {
		return nil, err
	}

	groupConsensusInfo := &pb.PoaGroupConsensusInfo{
		EpochDuration: DEFAULT_EPOCH_DURATION,
		Producers:     []string{ownerpubkey},
		CurrEpoch:     0,
		CurrBlockId:   0,
	}

	//initial PublicPoaGroupItem
	publicPoaGroup := &pb.PublicPOAGroupItem{
		AppKey:        params.AppKey,
		GroupId:       groupid,
		GroupName:     params.GroupName,
		OwnerPubKey:   ownerpubkey,
		SignPubkey:    ownerpubkey,
		CipherKey:     hex.EncodeToString(cipherKey),
		GenesisBlock:  genesisBlock,
		ConsensusInfo: groupConsensusInfo,
	}

	//marshal PublicPoaGroupItem
	publicPoaGroupByts, err := proto.Marshal(publicPoaGroup)
	if err != nil {
		return nil, err
	} //

	//create GroupItemRumLite
	groupItem := &pb.GroupItemRumLite{
		Type:      pb.GroupType_PUBLIC_POA,
		GroupData: publicPoaGroupByts,
	}

	//marshal GroupItemRumLite
	groupItemByts, err := proto.Marshal(groupItem)
	if err != nil {
		return nil, err
	}

	//create hash
	hash := localcrypto.Hash(groupItemByts)

	//create signature
	signature, err := ks.EthSignByKeyAlias(groupid, hash)
	if err != nil {
		return nil, err
	}

	//create GroupSeedRumLite
	groupSeed := &pb.GroupSeedRumLite{
		Group:     groupItem,
		Hash:      hash,
		Signature: signature,
	}

	result := &NewSeedResult{
		AppKey:  params.AppKey,
		GroupId: groupid,
		Seed:    groupSeed,
	}

	return result, nil
}

package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	"github.com/rumsystem/rumchaindata/pkg/pb"
)

type CreateGroupParam struct {
	GroupName      string `from:"group_name"      json:"group_name"      validate:"required"`
	ConsensusType  string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa"`
	EncryptionType string `from:"encryption_type" json:"encryption_type" validate:"required,oneof=public private"`
	AppKey         string `from:"app_key"         json:"app_key"         validate:"required,max=20,min=4"`
}

type GroupSeed struct {
	GenesisBlock   *pb.Block `json:"genesis_block" validate:"required"`
	GroupId        string    `json:"group_id" validate:"required"`
	GroupName      string    `json:"group_name" validate:"required"`
	OwnerPubkey    string    `json:"owner_pubkey" validate:"required"`
	ConsensusType  string    `json:"consensus_type" validate:"required,oneof=pos poa"`
	EncryptionType string    `json:"encryption_type" validate:"required,oneof=public private"`
	CipherKey      string    `json:"cipher_key" validate:"required"`
	AppKey         string    `json:"app_key" validate:"required"`
	Signature      string    `json:"signature" validate:"required"`
}

func CreateGroupUrl(params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (map[string]string, error) {
	result := make(map[string]string)
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	if params.ConsensusType != "poa" {
		return nil, errors.New("Other types of groups are not supported yet")
	}

	groupid := guuid.New()

	ks := nodectx.GetNodeCtx().Keystore

	/* init sign key */
	hexkey, err := initSignKey(groupid.String(), ks, nodeoptions)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}
	pubkeybytes, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	if err != nil {
		return nil, errors.New("UnmarshalSecp256k1PublicKey err:" + err.Error())
	}
	groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	genesisBlock, err := rumchaindata.CreateGenesisBlock(groupid.String(), p2ppubkey, ks, "")
	if err != nil {
		return nil, err
	}

	cipherKey, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}

	/* init encode key */
	groupEncryptPubkey, err := initEncryptKey(groupid.String(), ks)
	if err != nil {
		return nil, err
	}

	//create group item
	var item *pb.GroupItem
	item = &pb.GroupItem{}
	item.GroupId = groupid.String()
	item.GroupName = params.GroupName
	item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	item.UserSignPubkey = item.OwnerPubKey
	item.UserEncryptPubkey = groupEncryptPubkey
	item.ConsenseType = pb.GroupConsenseType_POA

	if params.EncryptionType == "public" {
		item.EncryptType = pb.GroupEncryptType_PUBLIC
	} else {
		item.EncryptType = pb.GroupEncryptType_PRIVATE
	}

	item.CipherKey = hex.EncodeToString(cipherKey)
	item.AppKey = params.AppKey
	item.HighestHeight = 0
	item.HighestBlockId = genesisBlock.BlockId
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = genesisBlock

	var group *chain.Group
	group = &chain.Group{}

	err = group.CreateGrp(item)
	if err != nil {
		return nil, err
	}

	if nodeoptions.IsRexTestMode == true {
		group.SetRumExchangeTestMode()
	}
	groupmgr := chain.GetGroupMgr()
	groupmgr.Groups[group.Item.GroupId] = group

	//create result
	encodedCipherKey := hex.EncodeToString(cipherKey)

	createGrpResult := &GroupSeed{
		GenesisBlock: genesisBlock, GroupId: groupid.String(),
		GroupName:      params.GroupName,
		OwnerPubkey:    item.OwnerPubKey,
		ConsensusType:  params.ConsensusType,
		EncryptionType: params.EncryptionType,
		CipherKey:      encodedCipherKey,
		AppKey:         params.AppKey,
		//Signature:      "", // updated by GenerateGroupSeedSignature
	}

	// generate signature
	//if err := GenerateGroupSeedSignature(createGrpResult); err != nil {
	//	return nil, err
	//}

	// save group seed to appdata
	pbGroupSeed := ToPbGroupSeed(*createGrpResult)
	if err := appdb.SetGroupSeed(&pbGroupSeed); err != nil {
		return nil, err
	}
	seedurl, err := GroupSeedToUrl(1, []string{"127.0.0.1:8080"}, createGrpResult)
	result["seed"] = seedurl
	result["group_id"] = groupid.String()
	return result, err
}

func CreateGroup(params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*GroupSeed, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	if params.ConsensusType != "poa" {
		return nil, errors.New("Other types of groups are not supported yet")
	}

	groupid := guuid.New()

	ks := nodectx.GetNodeCtx().Keystore

	/* init sign key */
	hexkey, err := initSignKey(groupid.String(), ks, nodeoptions)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}
	pubkeybytes, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	if err != nil {
		return nil, errors.New("UnmarshalSecp256k1PublicKey err:" + err.Error())
	}
	groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	genesisBlock, err := rumchaindata.CreateGenesisBlock(groupid.String(), p2ppubkey, ks, "")
	if err != nil {
		return nil, err
	}

	cipherKey, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}

	/* init encode key */
	groupEncryptPubkey, err := initEncryptKey(groupid.String(), ks)
	if err != nil {
		return nil, err
	}

	//create group item
	var item *pb.GroupItem
	item = &pb.GroupItem{}
	item.GroupId = groupid.String()
	item.GroupName = params.GroupName
	item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	item.UserSignPubkey = item.OwnerPubKey
	item.UserEncryptPubkey = groupEncryptPubkey
	item.ConsenseType = pb.GroupConsenseType_POA

	if params.EncryptionType == "public" {
		item.EncryptType = pb.GroupEncryptType_PUBLIC
	} else {
		item.EncryptType = pb.GroupEncryptType_PRIVATE
	}

	item.CipherKey = hex.EncodeToString(cipherKey)
	item.AppKey = params.AppKey
	item.HighestHeight = 0
	item.HighestBlockId = genesisBlock.BlockId
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = genesisBlock

	var group *chain.Group
	group = &chain.Group{}

	err = group.CreateGrp(item)
	if err != nil {
		return nil, err
	}

	if nodeoptions.IsRexTestMode == true {
		group.SetRumExchangeTestMode()
	}
	groupmgr := chain.GetGroupMgr()
	groupmgr.Groups[group.Item.GroupId] = group

	//create result
	encodedCipherKey := hex.EncodeToString(cipherKey)

	createGrpResult := &GroupSeed{
		GenesisBlock:   genesisBlock,
		GroupId:        groupid.String(),
		GroupName:      params.GroupName,
		OwnerPubkey:    item.OwnerPubKey,
		ConsensusType:  params.ConsensusType,
		EncryptionType: params.EncryptionType,
		CipherKey:      encodedCipherKey,
		AppKey:         params.AppKey,
		Signature:      "", // updated by GenerateGroupSeedSignature
	}

	// generate signature
	if err := GenerateGroupSeedSignature(createGrpResult); err != nil {
		return nil, err
	}

	// save group seed to appdata
	pbGroupSeed := ToPbGroupSeed(*createGrpResult)
	if err := appdb.SetGroupSeed(&pbGroupSeed); err != nil {
		return nil, err
	}

	return createGrpResult, nil
}

func GenerateGroupSeedSignature(result *GroupSeed) error {
	genesisBlockBytes, err := json.Marshal(result.GenesisBlock)
	if err != nil {
		e := fmt.Errorf("Marshal genesis block failed with msg: %s", err)
		return e
	}

	groupSignPubkey, err := p2pcrypto.ConfigDecodeKey(result.OwnerPubkey)
	if err != nil {
		e := fmt.Errorf("Decode group owner pubkey failed: %s", err)
		return e
	}

	cipherKey, err := hex.DecodeString(result.CipherKey)
	if err != nil {
		e := fmt.Errorf("Decode cipher key failed: %s", err)
		return e
	}

	var buffer bytes.Buffer
	buffer.Write(genesisBlockBytes)
	buffer.Write([]byte(result.GroupId))
	buffer.Write([]byte(result.GroupName))
	buffer.Write(groupSignPubkey) //group owner pubkey
	buffer.Write([]byte(result.ConsensusType))
	buffer.Write([]byte(result.EncryptionType))
	buffer.Write([]byte(result.AppKey))
	buffer.Write(cipherKey)

	hash := localcrypto.Hash(buffer.Bytes())
	ks := nodectx.GetNodeCtx().Keystore
	signature, err := ks.SignByKeyName(result.GroupId, hash)
	if err != nil {
		e := fmt.Errorf("ks.SignByKeyName failed: %s", err)
		return e
	}
	result.Signature = hex.EncodeToString(signature)

	return nil
}

// ToPbGroupSeed convert `api.GroupSeed` to `pb.GroupSeed`
func ToPbGroupSeed(s GroupSeed) pb.GroupSeed {
	return pb.GroupSeed{
		GenesisBlock:   s.GenesisBlock,
		GroupId:        s.GroupId,
		GroupName:      s.GroupName,
		OwnerPubkey:    s.OwnerPubkey,
		ConsensusType:  s.ConsensusType,
		EncryptionType: s.EncryptionType,
		CipherKey:      s.CipherKey,
		AppKey:         s.AppKey,
		Signature:      s.Signature,
	}
}

// FromPbGroupSeed convert `pb.GroupSeed` to `api.GroupSeed`
func FromPbGroupSeed(s *pb.GroupSeed) GroupSeed {
	return GroupSeed{
		GenesisBlock:   s.GenesisBlock,
		GroupId:        s.GroupId,
		GroupName:      s.GroupName,
		OwnerPubkey:    s.OwnerPubkey,
		ConsensusType:  s.ConsensusType,
		EncryptionType: s.EncryptionType,
		CipherKey:      s.CipherKey,
		AppKey:         s.AppKey,
		Signature:      s.Signature,
	}
}

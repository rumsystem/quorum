package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type CreateGroupParam struct {
	GroupName      string `from:"group_name"      json:"group_name"      validate:"required,max=20,min=5"`
	ConsensusType  string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa"`
	EncryptionType string `from:"encryption_type" json:"encryption_type" validate:"required,oneof=public private"`
	AppKey         string `from:"app_key"         json:"app_key"         validate:"required,max=20,min=5"`
}

type CreateGroupResult struct {
	GenesisBlock       *quorumpb.Block `json:"genesis_block" validate:"required"`
	GroupId            string          `json:"group_id" validate:"required"`
	GroupName          string          `json:"group_name" validate:"required,max=20,min=5"`
	OwnerPubkey        string          `json:"owner_pubkey" validate:"required"`
	OwnerEncryptPubkey string          `json:"owner_encryptpubkey" validate:"required"`
	ConsensusType      string          `json:"consensus_type" validate:"required,oneof=pos poa"`
	EncryptionType     string          `json:"encryption_type" validate:"required,oneof=public private"`
	CipherKey          string          `json:"cipher_key" validate:"required"`
	AppKey             string          `json:"app_key" validate:"required"`
	Signature          string          `json:"signature" validate:"required"`
}

func CreateGroup(params *CreateGroupParam, nodeoptions *options.NodeOptions) (*CreateGroupResult, error) {
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

	/* create genesis block */
	genesisBlock, err := chain.CreateGenesisBlock(groupid.String(), p2ppubkey)

	if err != nil {
		return nil, errors.New("Create genesis block failed with msg:" + err.Error())
	}

	genesisBlockBytes, err := json.Marshal(genesisBlock)
	if err != nil {
		return nil, errors.New("Marshal genesis block failed with msg:" + err.Error())
	}

	cipherKey, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}

	/* init encode key */
	groupEncryptPubkey, err := initEncodeKey(groupid.String(), ks)

	/* create group item */
	var item *quorumpb.GroupItem
	item = &quorumpb.GroupItem{}
	item.GroupId = groupid.String()
	item.GroupName = params.GroupName
	item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	item.UserSignPubkey = item.OwnerPubKey
	item.UserEncryptPubkey = groupEncryptPubkey
	item.ConsenseType = quorumpb.GroupConsenseType_POA

	if params.EncryptionType == "public" {
		item.EncryptType = quorumpb.GroupEncryptType_PUBLIC
	} else {
		item.EncryptType = quorumpb.GroupEncryptType_PRIVATE
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

	groupmgr := chain.GetGroupMgr()
	groupmgr.Groups[group.Item.GroupId] = group

	/* sign the result */
	var buffer bytes.Buffer
	buffer.Write(genesisBlockBytes)
	buffer.Write([]byte(groupid.String()))
	buffer.Write([]byte(params.GroupName))
	buffer.Write(groupSignPubkey) //group owner pubkey
	buffer.Write([]byte(params.ConsensusType))
	buffer.Write([]byte(params.EncryptionType))
	buffer.Write([]byte(params.AppKey))
	buffer.Write(cipherKey)

	hash := localcrypto.Hash(buffer.Bytes())
	signature, err := ks.SignByKeyName(groupid.String(), hash)
	encodedSign := hex.EncodeToString(signature)
	encodedCipherKey := hex.EncodeToString(cipherKey)

	createGrpResult := &CreateGroupResult{GenesisBlock: genesisBlock, GroupId: groupid.String(), GroupName: params.GroupName, OwnerPubkey: item.OwnerPubKey, OwnerEncryptPubkey: item.UserEncryptPubkey, ConsensusType: params.ConsensusType, EncryptionType: params.EncryptionType, CipherKey: encodedCipherKey, AppKey: params.AppKey, Signature: encodedSign}
	return createGrpResult, nil
}

package handlers

import (
	"encoding/hex"
	"errors"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	"github.com/rumsystem/quorum/pkg/pb"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type NewGroupSeedParams struct {
	AppId              string `from:"app_id"                    json:"app_id"                    validate:"required"`
	AppName            string `from:"app_name"                  json:"app_name"                  validate:"required"`
	GroupName          string `from:"group_name"                json:"group_name"                validate:"required" example:"demo app"`
	ConsensusType      string `from:"consensus_type"            json:"consensus_type"            validate:"required,oneof=pos poa" example:"poa"`
	SyncType           string `from:"sync_type"                 json:"sync_type"                 validate:"required,oneof=public private" example:"public"`
	OwnerKeyName       string `from:"owner_keyname"             json:"owner_keyname"             example:"group_owner_key_name"`
	NeoProducerKeyName string `from:"neoproducer_sign_keyname"  json:"neoproducer_sign_keyname"  example:"general_producer_pubkey_name"`
	EpochDuration      int64  `from:"epoch_duration"            json:"epoch_duration"            validate:"required" example:"1000"` //ms
	Url                string `from:"url"                       json:"url"                       example:"https://www.rumdemo.com"`  //point to somewhere, like app website
}

type NewGroupSeedResult struct {
	GroupId         string        `json:"group_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	OwnerKeyName    string        `json:"owner_keyname" validate:"required" example:"group_owner_key_name"`
	ProducerKeyName string        `json:"producer_sign_keyname" validate:"required" example:"general_producer_pubkey_name"`
	Seed            *pb.GroupSeed `json:"seed" validate:"required"`
	SeedByts        []byte        `json:"seed_byts" validate:"required"`
}

const DEFAULT_EPOCH_DURATION = 1000 //m
const NEOPROUDCER_SIGNKEY_SURFIX = "_neoproducer_sign_keyname"

func NewGroupSeed(params *NewGroupSeedParams, nodeoptions *options.NodeOptions) (*NewGroupSeedResult, error) {
	if params.ConsensusType != "poa" {
		return nil, errors.New("consensus_type must be poa, other types are not supported in rum-lite")
	}

	var syncType quorumpb.GroupSyncType
	if params.SyncType != "public" {
		syncType = quorumpb.GroupSyncType_PUBLIC
	} else if params.SyncType != "private" {
		syncType = quorumpb.GroupSyncType_PRIVATE
	} else {
		return nil, errors.New("sync_type must be public or private")
	}

	groupid := guuid.New().String()
	ks := localcrypto.GetKeystore()

	var ownerKeyName, ownerPubkey string
	var err error
	if params.OwnerKeyName == "" {
		//init ownerpubkey
		ownerKeyName = groupid
		ownerPubkey, err = localcrypto.InitSignKeyWithKeyName(ownerKeyName, nodeoptions)
		if err != nil {
			return nil, errors.New("initial group owner keypair failed, err:" + err.Error())
		}
	} else {
		ownerKeyName = params.OwnerKeyName
		ownerPubkey, err = ks.GetEncodedPubkey(ownerKeyName, localcrypto.Sign)
		if err != nil {
			return nil, errors.New("owner_keyname not found in local keystore")
		}
	}

	var producerPubkey string
	var producerKeyName string
	var producers []string
	if params.NeoProducerKeyName != "" {
		producerKeyName = params.NeoProducerKeyName
		producerPubkey, err = ks.GetEncodedPubkey(producerKeyName, localcrypto.Sign)
		if err != nil {
			return nil, errors.New("producer_sign_keyname not found in local keystore")
		}
		producers = append(producers, producerPubkey)
	} else {
		producerKeyName = groupid + NEOPROUDCER_SIGNKEY_SURFIX
		producerPubkey, err = localcrypto.InitSignKeyWithKeyName(producerKeyName, nodeoptions)
		if err != nil {
			return nil, errors.New("initial group producer sign key failed, err:" + err.Error())
		}
		producers = append(producers, producerPubkey)
	}

	//init cipher key
	cipherKeyBytes, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}
	cipherKey := hex.EncodeToString(cipherKeyBytes)
	if err != nil {
		return nil, err
	}

	//init fork info
	forkItem := &pb.ForkItem{
		GroupId:        groupid,
		StartFromBlock: 0,
		StartFromEpoch: 0,
		EpochDuration:  uint64(params.EpochDuration),
		Producers:      producers,
		Memo:           "Initial Fork",
	}

	poaConsensusInfo := &pb.PoaConsensusInfo{
		ConsensusId: guuid.New().String(),
		ChainVer:    0,
		InTrx:       "",
		ForkInfo:    forkItem,
	}

	//hash consensus info
	consensusInfoByts, err := proto.Marshal(poaConsensusInfo)
	if err != nil {
		return nil, err
	}

	//create Consensus
	consensus := &pb.Consensus{
		Type: pb.GroupConsenseType_POA,
		Data: consensusInfoByts,
	}

	//create genesis block
	genesisBlock, err := rumchaindata.CreateGenesisBlockByEthKey(groupid, consensus, producerKeyName)
	if err != nil {
		return nil, err
	}

	//create group seed

	/*
	   message GroupSeed {
	       Block  GenesisBlock   = 1;
	       string GroupId        = 2;
	       string GroupName      = 3;
	       string OwnerPubkey    = 4;
	       string ConsensusType  = 5;
	       string SyncType       = 6;
	       string CipherKey      = 7;
	       string AppKey         = 8;
	       bytes  Hash           = 9;
	       bytes  Signature      = 10;
	   }
	*/

	groupSeed := &pb.GroupSeed{
		GenesisBlock: genesisBlock,
		GroupId:      groupid,
		GroupName:    params.GroupName,
		OwnerPubkey:  ownerPubkey,
		SyncType:     syncType,
		CipherKey:    cipherKey,
		AppId:        params.AppId,
		AppName:      params.AppName,
		Hash:         nil,
		Signature:    nil,
	}

	//hash groupItem
	seedByts, err := proto.Marshal(groupSeed)
	if err != nil {
		return nil, err
	}

	//sign hash by owner key
	hash := localcrypto.Hash(seedByts)
	signature, err := localcrypto.GetKeystore().EthSignByKeyName(ownerKeyName, hash)

	if err != nil {
		return nil, err
	}

	groupSeed.Hash = hash
	groupSeed.Signature = signature

	seedBytsWithSign, err := proto.Marshal(groupSeed)
	if err != nil {
		return nil, err
	}

	return &NewGroupSeedResult{
		GroupId:         groupid,
		OwnerKeyName:    ownerKeyName,
		ProducerKeyName: producerKeyName,
		Seed:            groupSeed,
		SeedByts:        seedBytsWithSign,
	}, nil
}

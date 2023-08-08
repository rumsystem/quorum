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
	AppName            string `from:"app_name"                  json:"app_name"                  validate:"required,max=100,min=2" example:"demo app"`
	ConsensusType      string `from:"consensus_type"            json:"consensus_type"            validate:"required,oneof=pos poa" example:"poa"`
	SyncType           string `from:"sync_type"                 json:"sync_type"                 validate:"required,oneof=public private" example:"public"`
	EncryptTrx         bool   `from:"encrypt_trx"               json:"encrypt_trx"               validate:"required" example:"true"`
	CtnType            string `from:"ctn_type"                  json:"ctn_type"                  validate:"required,oneof=blob service" example:"blob"`
	OwnerKeyName       string `from:"owner_keyname"             json:"owner_keyname"             example:"group_owner_key_name"`
	NeoProducerKeyName string `from:"neoproducer_sign_keyname"  json:"neoproducer_sign_keyname"  example:"general_producer_pubkey_name"`
	EpochDuration      int64  `from:"epoch_duration"            json:"epoch_duration"            validate:"required" example:"1000"` //ms
	Url                string `from:"url"                       json:"url"                       example:"https://www.rumdemo.com"`  //point to somewhere, like app website
}

type NewGroupSeedResult struct {
	GroupId         string               `json:"group_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	OwnerKeyName    string               `json:"owner_keyname" validate:"required" example:"group_owner_key_name"`
	ProducerKeyName string               `json:"producer_sign_keyname" validate:"required" example:"general_producer_pubkey_name"`
	Seed            *pb.GroupSeedRumLite `json:"seed" validate:"required"`
	SeedByts        []byte               `json:"seed_byts" validate:"required"`
}

const DEFAULT_EPOCH_DURATION = 1000 //ms
const NEOPROUDCER_SIGNKEY_SURFIX = "_neoproducer_sign_keyname"

func NewGroupSeed(params *NewGroupSeedParams, nodeoptions *options.NodeOptions) (*NewGroupSeedResult, error) {
	var consensusType quorumpb.GroupConsenseType
	if params.ConsensusType != "poa" {
		return nil, errors.New("consensus_type must be poa, other types are not supported in rum-lite")
	}
	consensusType = quorumpb.GroupConsenseType_POA

	var syncType quorumpb.GroupSyncType
	if params.SyncType != "public" {
		syncType = quorumpb.GroupSyncType_PUBLIC_SYNC
	} else if params.SyncType != "private" {
		syncType = quorumpb.GroupSyncType_PRIVATE_SYNC
	} else {
		return nil, errors.New("sync_type must be public or private")
	}

	var ctnType quorumpb.GroupCtnType
	if params.CtnType == "blob" {
		ctnType = quorumpb.GroupCtnType_BLOB
	} else if params.CtnType == "service" {
		ctnType = quorumpb.GroupCtnType_SERVICE
	} else {
		return nil, errors.New("chain_type must be \"blob\" or \"service\"")
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

	var cipherKey string
	if params.EncryptTrx {
		//init cipher key
		cipherKeyBytes, err := localcrypto.CreateAesKey()
		if err != nil {
			return nil, err
		}
		cipherKey = hex.EncodeToString(cipherKeyBytes)
		if err != nil {
			return nil, err
		}
	} else {
		cipherKey = ""
	}

	poaConsensusInfo := &pb.POAConsensusInfo{
		ConsensusId:   guuid.New().String(),
		EpochDuration: uint64(params.EpochDuration),
		Producers:     producers,
		CurrEpoch:     0,
		CurrBlockId:   0,
	}

	consensusInfo := &pb.ConsensusInfoRumLite{
		Poa: poaConsensusInfo,
	}

	//create genesis block
	genesisBlock, err := rumchaindata.CreateGenesisBlockRumLiteByEthKey(groupid, producerPubkey, producerKeyName, consensusInfo)
	if err != nil {
		return nil, err
	}

	//create GroupItemRumLite
	groupItem := &pb.GroupItemRumLite{
		AppId:         params.AppId,
		AppName:       params.AppName,
		GroupId:       groupid,
		OwnerPubKey:   ownerPubkey,
		TrxSignPubkey: "",
		EncryptTrxCtn: params.EncryptTrx,
		CipherKey:     cipherKey,
		SyncType:      syncType,
		CtnType:       ctnType,
		ConsenseType:  consensusType,
		ConsensusInfo: consensusInfo,
		GenesisBlock:  genesisBlock,
		LastUpdate:    genesisBlock.TimeStamp,
	}

	//hash groupItem
	groupItemByts, err := proto.Marshal(groupItem)
	if err != nil {
		return nil, err
	}

	//sign hash by owner key (groupId)
	hash := localcrypto.Hash(groupItemByts)
	signature, err := localcrypto.GetKeystore().EthSignByKeyName(ownerKeyName, hash)

	if err != nil {
		return nil, err
	}

	seed := &pb.GroupSeedRumLite{
		Group:     groupItem,
		Hash:      hash,
		Signature: signature,
	}

	seedByts, err := proto.Marshal(seed)
	if err != nil {
		return nil, err
	}

	return &NewGroupSeedResult{
		GroupId:         groupid,
		OwnerKeyName:    ownerKeyName,
		ProducerKeyName: producerKeyName,
		Seed:            seed,
		SeedByts:        seedByts,
	}, nil
}

package handlers

import (
	"encoding/hex"
	"errors"

	"github.com/go-playground/validator/v10"
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
	ChainType          string `from:"chain_type"                json:"chain_type"                validate:"required,oneof=archive dynamic" example:"archive"`
	OwnerKeyName       string `from:"owner_keyname"             json:"owner_keyname"             example:"group_owner_key_name"`
	TrxSignKeyName     string `from:"trx_sign_keyname"          json:"trx_sign_keyname"          example:"my_sign_pubkey_name"`
	NeoProducerKeyName string `from:"neoproducer_sign_keyname"  json:"neoproducer_sign_keyname"  example:"general_producer_pubkey_name"`
	EpochDuration      int64  `from:"epoch_duration"            json:"epoch_duration"            validate:"required" example:"1000"` //ms
	Url                string `from:"url"                       json:"url"                       example:"https://www.rumdemo.com"`  //point to somewhere, like app website
}

type NewGroupSeedResult struct {
	GroupId         string               `json:"group_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	OwnerKeyName    string               `json:"owner_keyname" validate:"required" example:"group_owner_key_name"`
	TrxSignKeyName  string               `json:"trx_sign_keyname" validate:"required" example:"my_sign_pubkey_name"`
	ProducerKeyName string               `json:"producer_sign_keyname" validate:"required" example:"general_producer_pubkey_name"`
	Seed            *pb.GroupSeedRumLite `json:"seed" validate:"required"`
	SeedByts        []byte               `json:"seed_byts" validate:"required"`
}

const DEFAULT_EPOCH_DURATION = 1000 //ms
const TRX_SIGNKEY_SURFIX = "_trx_sign_keyname"
const NEOPROUDCER_SIGNKEY_SURFIX = "_neoproducer_sign_keyname"

func NewGroupSeed(params *NewGroupSeedParams, nodeoptions *options.NodeOptions) (*NewGroupSeedResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

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

	var chainType quorumpb.GroupChainType
	if params.ChainType == "archive" {
		chainType = quorumpb.GroupChainType_ARCHIVE_CHAIN
	} else if params.ChainType == "dynamic" {
		chainType = quorumpb.GroupChainType_DYNAMIC_CHAIN
	} else {
		return nil, errors.New("chain_type must be archive or dynamic")
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

	var trxSignKeyName, trxSignPubkey string
	if params.TrxSignKeyName == "" {
		//init trx sign key
		trxSignKeyName = groupid + TRX_SIGNKEY_SURFIX
		trxSignPubkey, err = localcrypto.InitSignKeyWithKeyName(trxSignKeyName, nodeoptions)
		if err != nil {
			return nil, errors.New("initial group trx sign keypair failed, err:" + err.Error())
		}
	} else {
		trxSignKeyName = params.TrxSignKeyName
		trxSignPubkey, err = ks.GetEncodedPubkey(trxSignKeyName, localcrypto.Sign)
		if err != nil {
			return nil, errors.New("trx_sign_keyname not found in local keystore")
		}
	}

	var producers []string
	var producerKeyName string
	if params.NeoProducerKeyName != "" {
		producerKeyName = params.NeoProducerKeyName
		producerPubkey, err := ks.GetEncodedPubkey(producerKeyName, localcrypto.Sign)
		if err != nil {
			return nil, errors.New("producer_sign_keyname not found in local keystore")
		}
		producers = append(producers, producerPubkey)
	} else {
		producerKeyName = groupid + NEOPROUDCER_SIGNKEY_SURFIX
		producerPubkey, err := localcrypto.InitSignKeyWithKeyName(producerKeyName, nodeoptions)
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
	genesisBlock, err := rumchaindata.CreateGenesisBlockRumLiteByEthKey(groupid, ownerPubkey, ownerKeyName, consensusInfo)
	if err != nil {
		return nil, err
	}

	//create GroupItemRumLite
	groupItem := &pb.GroupItemRumLite{
		AppId:         params.AppId,
		AppName:       params.AppName,
		GroupId:       groupid,
		OwnerPubKey:   ownerPubkey,
		TrxSignPubkey: trxSignPubkey,
		EncryptTrx:    params.EncryptTrx,
		CipherKey:     cipherKey,
		SyncType:      syncType,
		ChainType:     chainType,
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
		TrxSignKeyName:  trxSignKeyName,
		ProducerKeyName: producerKeyName,
		Seed:            seed,
		SeedByts:        seedByts,
	}, nil
}

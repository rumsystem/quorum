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

type NewSeedParams struct {
	AppId         string `from:"app_id"          json:"app_id"          validate:"required"` //uuid
	AppName       string `from:"app_name"        json:"app_name"        validate:"required,max=100,min=2" example:"demo app"`
	ConsensusType string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa" example:"poa"`
	SyncType      string `from:"sync_type"       json:"sync_type"       validate:"required,oneof=public private" example:"public"`
	EncryptTrx    bool   `from:"encrypt_trx"     json:"encrypt_trx"     validate:"required" example:"true"`
	Url           string `from:"url"             json:"url"             example:"https://www.rumdemo.com"` //point to somewhere, like app website
}

type NewSeedResult struct {
	GroupId string `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	Seed    []byte `json:"seed" validate:"required"`
}

const DEFAULT_EPOCH_DURATION = 1000 //ms
const NEO_PRODUCER_SIGNKEY_SURFIX = "_neo_producer_signkey"

func NewSeed(params *NewSeedParams, nodeoptions *options.NodeOptions) (*NewSeedResult, error) {
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
	} else {
		syncType = quorumpb.GroupSyncType_PRIVATE_SYNC
	}

	//create groupid
	groupid := guuid.New().String()

	//init keystore
	ks := localcrypto.GetKeystore()

	//init ownerpubkey
	ownerpubkey, err := localcrypto.InitSignKeyWithKeyName(groupid, nodeoptions)
	if err != nil {
		return nil, errors.New("initial group owner keypair failed, err:" + err.Error())
	}

	//initial first producer pubkey
	neoProducerPubkey, err := localcrypto.InitSignKeyWithKeyName(groupid+NEO_PRODUCER_SIGNKEY_SURFIX, nodeoptions)
	if err != nil {
		return nil, errors.New("initial group  neo producer keypari failed, err:" + err.Error())
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
		ChainVer:      0,
		EpochDuration: DEFAULT_EPOCH_DURATION,
		Producers:     []string{neoProducerPubkey},
		CurrEpoch:     0,
		CurrBlockId:   0,
	}

	consensusInfo := &pb.ConsensusInfoRumLite{
		Poa: poaConsensusInfo,
	}

	//create genesis block
	genesisBlock, err := rumchaindata.CreateGenesisBlockRumLiteByEthKey(groupid, neoProducerPubkey, consensusInfo, ks, neoProducerPubkey)
	if err != nil {
		return nil, err
	}

	//create GroupItemRumLite
	groupItem := &pb.GroupItemRumLite{
		AppId:   params.AppId,
		AppName: params.AppName,
		GroupId: groupid,

		OwnerPubKey:    ownerpubkey,
		UserSignPubkey: ownerpubkey,
		EncryptTrx:     params.EncryptTrx,
		CipherKey:      cipherKey,
		SyncType:       syncType,
		ConsenseType:   consensusType,
		ConsensusInfo:  consensusInfo,
		GenesisBlock:   genesisBlock,
		LastUpdate:     genesisBlock.TimeStamp,
	}

	//hash groupItem
	groupItemByts, err := proto.Marshal(groupItem)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(groupItemByts)

	//sign hash by owner key (groupId)
	signature, err := ks.EthSignByKeyAlias(groupid, hash)

	seed := &pb.GroupSeedRumLite{
		Group:     groupItem,
		Hash:      hash,
		Signature: signature,
	}

	seedByts, err := proto.Marshal(seed)
	if err != nil {
		return nil, err
	}

	return &NewSeedResult{
		GroupId: groupid,
		Seed:    seedByts,
	}, nil
}

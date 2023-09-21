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

type SyncServcieParams struct {
	SSyncerKeyname string `from:"syncer_keyname"  json:"syncer_keyname"`
	Term           string `from:"term"              json:"term"              validate:"required"`
	Contract       []byte `from:"contract"          json:"contract"          validate:"required"`
}

type ProduceServiceParams struct {
	SProducerKeyName string `from:"producer_keyname"  json:"producer_keyname"`
	Term             string `from:"term"           json:"term"            validate:"required"`
	Contract         []byte `from:"contract"       json:"contract"        validate:"required"`
}

type CtnServiceParams struct {
	Apis     []string `from:"apis"              json:"apis"              validate:"required"`
	Term     string   `from:"term"              json:"term"              validate:"required"`
	Contract []byte   `from:"contract"          json:"contract"          validate:"required"`
}

type PublishServiceParams struct {
	Term     string `from:"term"            json:"term"            validate:"required"`
	Contract []byte `from:"contract"        json:"contract"        validate:"required"`
}

type NewGroupSeedParams struct {
	AppId         string `from:"app_id"                    json:"app_id"                    validate:"required"`
	AppName       string `from:"app_name"                  json:"app_name"                  validate:"required"`
	GroupName     string `from:"group_name"                json:"group_name"                validate:"required" example:"demo app"`
	ConsensusType string `from:"consensus_type"            json:"consensus_type"            validate:"required,oneof=pos poa" example:"poa"`
	AuthType      string `from:"auth_type"                 json:"auth_type"                 validate:"required,oneof=public private" example:"public"`
	EpochDuration int64  `from:"epoch_duration"            json:"epoch_duration"            validate:"required" example:"1000"` //ms

	OwnerKeyName    string `from:"owner_keyname"             json:"owner_keyname"             example:"group_owner_key_name"`
	ProducerKeyName string `from:"producer_keyname"          json:"producer_keyname"          example:"general_producer_pubkey_name"`

	Url string `from:"url"                       json:"url"                       example:"https://www.rumdemo.com"` //point to somewhere, like app website

	//for group service
	PublishService *PublishServiceParams `from:"publish_service"  json:"publish_service"`
	SyncService    *SyncServcieParams    `from:"sync_service"     json:"sync_service"`
	ProduceService *ProduceServiceParams `from:"produce_service"  json:"produce_service"`
	CtnService     *CtnServiceParams     `from:"ctn_service"      json:"ctn_service"`
}

type NewGroupSeedResult struct {
	GroupId                string   `json:"group_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	OwnerKeyName           string   `json:"owner_keyname" validate:"required" example:"group_owner_key_name"`
	GroupProducerKeyName   string   `json:"group_producer_keyname" validate:"required" example:"local_group_producer_keyname"`
	ServiceProducerKeyname string   `json:"service_producer_keyname" validate:"required" example:"service_producer_keyname"`
	ServiceSyncerkeyname   string   `json:"service_syncer_keyname" validate:"required" example:"service_syncer_pubkey_name"`
	ServicePosterKeyname   string   `json:"service_poster_keyname" validate:"required" example:"service_poster_pubkey_name"`
	CtnAPIs                []string `json:"ctn_apis" validate:"required" example:"[\"/api/v1/xxx\",\"/api/v1/yyy\"]"`
	SeedByts               []byte   `json:"seed" validate:"required"`
}

const DEFAULT_EPOCH_DURATION = 1000 //m
const NEOPROUDCER_KEYNAME_SURFIX = "_np_kn"
const DEFAULT_SERVICE_SYNCER_KEYNAME_SURFIX = "_ss_kn"
const DEFAULT_SERVICE_PRODUCER_KEYNAME_SURFIX = "_spp_kn"
const DEFAULT_SERVICE_POSTER_KEYNAME_SURFIX = "_spt_kn"

func NewGroupSeed(params *NewGroupSeedParams, nodeoptions *options.NodeOptions) (*NewGroupSeedResult, error) {
	if params.ConsensusType != "poa" {
		return nil, errors.New("consensus_type must be poa, other types are not supported in rum-lite")
	}

	var authType quorumpb.GroupAuthType
	if params.AuthType == "public" {
		authType = quorumpb.GroupAuthType_PUBLIC
	} else if params.AuthType == "private" {
		authType = quorumpb.GroupAuthType_PRIVATE
	} else {
		return nil, errors.New("auth_type must be public or private")
	}

	ks := localcrypto.GetKeystore()
	groupid := guuid.New().String()

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
	if params.ProducerKeyName != "" {
		producerKeyName = params.ProducerKeyName
		producerPubkey, err = ks.GetEncodedPubkey(producerKeyName, localcrypto.Sign)
		if err != nil {
			return nil, errors.New("producer_sign_keyname not found in local keystore")
		}
		producers = append(producers, producerPubkey)
	} else {
		producerKeyName = groupid + NEOPROUDCER_KEYNAME_SURFIX
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
	genesisBlock, err := rumchaindata.CreateGenesisBlockByEthKey(groupid, consensus, producerPubkey, producerKeyName)
	if err != nil {
		return nil, err
	}

	sProducerKeyname := ""
	sProducerPubkey := ""
	sSyncerKeyname := ""
	sSyncerPubkey := ""

	produceService, syncService, ctnService, publishService := false, false, false, false

	if params.ProduceService != nil {
		produceService = true
	}

	if params.SyncService != nil {
		syncService = true
	}

	if params.CtnService != nil {
		ctnService = true
	}

	if params.PublishService != nil {
		publishService = true
	}

	//if produce service is provided , then sync service is also needed
	if produceService && !syncService {
		return nil, errors.New("sync_service is needed if produce_service is provided")
	}

	if ctnService && !syncService {
		return nil, errors.New("sync_service is needed if ctn_service is provided")
	}

	needSProducerKey, needSSyncerKey := false, false

	if syncService {
		needSSyncerKey = true
		if produceService {
			needSProducerKey = true
		}
	}

	if needSProducerKey {
		if params.ProduceService.SProducerKeyName == "" {
			sProducerKeyname = groupid + DEFAULT_SERVICE_PRODUCER_KEYNAME_SURFIX
			sProducerPubkey, err = localcrypto.InitSignKeyWithKeyName(sProducerKeyname, nodeoptions)
			if err != nil {
				return nil, errors.New("initial group producer sign key failed, err:" + err.Error())
			}
		} else {
			sProducerKeyname = params.ProduceService.SProducerKeyName
			sProducerPubkey, err = ks.GetEncodedPubkey(sProducerKeyname, localcrypto.Sign)
			if err != nil {
				return nil, errors.New("brewer_keyname not found in local keystore")
			}
		}
	}

	if needSSyncerKey {
		if params.SyncService.SSyncerKeyname == "" {
			sSyncerKeyname = groupid + DEFAULT_SERVICE_SYNCER_KEYNAME_SURFIX
			sSyncerPubkey, err = localcrypto.InitSignKeyWithKeyName(sSyncerKeyname, nodeoptions)
			if err != nil {
				return nil, errors.New("initial group producer sign key failed, err:" + err.Error())
			}
		} else {
			sSyncerKeyname = params.SyncService.SSyncerKeyname
			sSyncerPubkey, err = ks.GetEncodedPubkey(sSyncerKeyname, localcrypto.Sign)
			if err != nil {
				return nil, errors.New("syncer_keyname not found in local keystore")
			}
		}
	}

	//create group services
	groupServices := []*pb.GroupService{}

	if produceService {
		producer := &pb.Producer{
			GroupId:        groupid,
			ProducerPubkey: sProducerPubkey,
			Memo:           "Service producer",
		}

		produceService := &pb.ProduceServiceItem{
			Producer: producer,
			Term:     params.ProduceService.Term,
			Contract: params.ProduceService.Contract,
		}

		produceServiceByts, err := proto.Marshal(produceService)
		if err != nil {
			return nil, err
		}

		service := &pb.GroupService{
			TaskType: pb.GroupTaskType_PRODUCE,
			Data:     produceServiceByts,
		}
		groupServices = append(groupServices, service)
	}

	if syncService {
		syncer := &pb.Syncer{
			GroupId:      groupid,
			SyncerPubkey: sSyncerPubkey,
			Memo:         "Service syncer",
		}

		syncService := &pb.SyncServiceItem{
			Syncer:   syncer,
			Term:     params.SyncService.Term,
			Contract: params.SyncService.Contract,
		}

		syncServiceByts, err := proto.Marshal(syncService)
		if err != nil {
			return nil, err
		}

		service := &pb.GroupService{
			TaskType: pb.GroupTaskType_SYNC,
			Data:     syncServiceByts,
		}

		groupServices = append(groupServices, service)
	}

	if ctnService {
		ctnService := &pb.CtnServiceItem{
			APIs:     params.CtnService.Apis,
			Term:     params.CtnService.Term,
			Contract: params.CtnService.Contract,
		}

		ctnServiceByts, err := proto.Marshal(ctnService)
		if err != nil {
			return nil, err
		}

		service := &pb.GroupService{
			TaskType: pb.GroupTaskType_CTN,
			Data:     ctnServiceByts,
		}

		groupServices = append(groupServices, service)
	}

	if publishService {
		publishService := &pb.PublishServiceItem{
			Term:     params.PublishService.Term,
			Contract: params.PublishService.Contract,
		}

		postServiceByts, err := proto.Marshal(publishService)
		if err != nil {
			return nil, err
		}

		service := &pb.GroupService{
			TaskType: pb.GroupTaskType_PUBLISH,
			Data:     postServiceByts,
		}
		groupServices = append(groupServices, service)
	}

	groupSeed := &pb.GroupSeed{
		GenesisBlock: genesisBlock,
		GroupId:      groupid,
		GroupName:    params.GroupName,
		OwnerPubkey:  ownerPubkey,
		AuthType:     authType,
		CipherKey:    cipherKey,
		AppId:        params.AppId,
		AppName:      params.AppName,
		Services:     groupServices,
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
		GroupId:                groupid,
		OwnerKeyName:           ownerKeyName,
		GroupProducerKeyName:   producerKeyName,
		ServiceProducerKeyname: sProducerKeyname,
		ServiceSyncerkeyname:   sSyncerKeyname,
		CtnAPIs:                params.CtnService.Apis,
		SeedByts:               seedBytsWithSign,
	}, nil
}

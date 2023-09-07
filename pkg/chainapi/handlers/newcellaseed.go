package handlers

import (
	"encoding/hex"
	"errors"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	"github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type NewCellarSeedParams struct {
	CellarName         string `from:"cella_name"                json:"cella_name"                validate:"required"`
	OwnerKeyName       string `from:"owner_keyname"             json:"owner_keyname"             example:"group_owner_key_name"`
	BrewerKeyName      string `from:"brewer_keyname"            json:"brewer_keyname"            example:"general_brewer_pubkey_name"`
	ProducerKeyName    string `from:"producer_keyname"          json:"producer_keyname"          example:"general_producer_pubkey_name"`
	EpochDuration      int64  `from:"epoch_duration"            json:"epoch_duration"            validate:"required" example:"1000"` //ms
	BrewServiceTerm    string `from:"brew_service_term"         json:"brew_service_term"         validate:"required"`
	StorageServiceTerm string `from:"storage_service_term"      json:"storage_service_term"      validate:"required"`
	Memo               string `from:"memo"                      json:"memo"                      example:"cella memo"`
}

type NewCellarSeedResult struct {
	CellarId        string         `json:"cella_id" validate:"required" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	OwnerKeyName    string         `json:"owner_keyname" validate:"required" example:"group_owner_key_name"`
	BrewerkeyName   string         `json:"brewer_keyname" validate:"required" example:"general_brewer_pubkey_name"`
	ProducerKeyName string         `json:"producer_keyname" validate:"required" example:"general_producer_pubkey_name"`
	Seed            *pb.CellarSeed `json:"seed" validate:"required"`
	SeedByts        []byte         `json:"seed_byts" validate:"required"`
}

const CELLA_BREWER_SIGNKEY_SURFIX = "_brewer_sign_keyname"

func NewCellarSeed(params *NewCellarSeedParams, nodeoptions *options.NodeOptions) (*NewCellarSeedResult, error) {
	cellarid := guuid.New().String()
	ks := localcrypto.GetKeystore()

	//create ceall group
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

	var brewerKeyNanme, brewerPubkey string
	if params.BrewerKeyName == "" {
		brewerKeyNanme = cellarid + CELLA_BREWER_SIGNKEY_SURFIX
		brewerPubkey, err = localcrypto.InitSignKeyWithKeyName(brewerKeyNanme, nodeoptions)
		if err != nil {
			return nil, errors.New("initial group brewer keypair failed, err:" + err.Error())
		}
	} else {
		brewerKeyNanme = params.BrewerKeyName
		brewerPubkey, err = ks.GetEncodedPubkey(brewerKeyNanme, localcrypto.Sign)
		if err != nil {
			return nil, errors.New("brewer_keyname not found in local keystore")
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
	genesisBlock, err := rumchaindata.CreateGenesisBlockByEthKey(groupid, consensus, producerPubkey, producerKeyName)
	if err != nil {
		return nil, err
	}

	groupSeed := &pb.GroupSeed{
		GenesisBlock: genesisBlock,
		GroupId:      groupid,
		GroupName:    params.CellarName + "_group",
		OwnerPubkey:  ownerPubkey,
		SyncType:     pb.GroupSyncType_PRIVATE,
		CipherKey:    cipherKey,
		AppId:        "",
		AppName:      "",
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

	//create cella seed

	storageServiceTerm := &pb.StorageServiceTermItem{
		Term: params.StorageServiceTerm,
	}

	ssByts, err := proto.Marshal(storageServiceTerm)
	if err != nil {
		return nil, err
	}

	csst := &pb.CellarServiceTermItem{
		Type: pb.CellarServiceType_STORAGE,
		Term: ssByts,
	}

	brewServiceTerm := &pb.BrewServiceTermItem{
		Term: params.BrewServiceTerm,
	}

	bsByts, err := proto.Marshal(brewServiceTerm)
	if err != nil {
		return nil, err
	}

	bsst := &pb.CellarServiceTermItem{
		Type: pb.CellarServiceType_BREW,
		Term: bsByts,
	}

	cellarSeed := &pb.CellarSeed{
		CellarId:             cellarid,
		CellarName:           params.CellarName,
		CellarOwnerPubkey:    ownerPubkey,
		CellarBrewerPubkey:   brewerPubkey,
		CellarProducerPubkey: producerPubkey,
		ServiceTerms:         []*pb.CellarServiceTermItem{csst, bsst},
		CellarGroupSeed:      groupSeed,
		Hash:                 nil,
		Signature:            nil,
	}

	cellarByts, err := proto.Marshal(cellarSeed)
	if err != nil {
		return nil, err
	}

	hashcellar := localcrypto.Hash(cellarByts)
	cellarSign, err := localcrypto.GetKeystore().EthSignByKeyName(ownerKeyName, hashcellar)
	if err != nil {
		return nil, err
	}

	cellarSeed.Hash = hashcellar
	cellarSeed.Signature = cellarSign

	cellarSeedBytsWithSign, err := proto.Marshal(cellarSeed)
	if err != nil {
		return nil, err
	}

	return &NewCellarSeedResult{
		CellarId:        cellarid,
		OwnerKeyName:    ownerKeyName,
		BrewerkeyName:   brewerKeyNanme,
		ProducerKeyName: producerKeyName,
		Seed:            cellarSeed,
		SeedByts:        cellarSeedBytsWithSign,
	}, nil

}

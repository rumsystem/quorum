package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/rumsystem/quorum/pkg/pb"
)

type CreateGroupParam struct {
	GroupName       string `from:"group_name"      json:"group_name"      validate:"required,max=100,min=2" example:"demo group"`
	ConsensusType   string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa" example:"poa"`
	EncryptionType  string `from:"encryption_type" json:"encryption_type" validate:"required,oneof=public private" example:"public"`
	AppKey          string `from:"app_key"         json:"app_key"         validate:"required,max=20,min=4" example:"test_app"`
	IncludeChainUrl bool   `json:"include_chain_url" example:"true"`
}

type JoinGroupParamV2 struct {
	Seed string `json:"seed" validate:"required" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"` // seed url
}

type GroupSeed struct {
	/* Example: {
	       "BlockId": "80e3dbd6-24de-46cd-9290-ed2ae93ec3ac",
	       "GroupId": "c0020941-e648-40c9-92dc-682645acd17e",
	       "ProducerPubKey": "CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg==",
	       "Hash": "LOZa0CLITIpuQqpvXb6LyXV9z+2rSoU4JwBq0BCXttc=",
	       "Signature": "MEQCICAXCicQ6f4hRNSoJR89DF3a6AKpe6ZgLXsjXqH9H3jxAiA8dpukcriwEu8amouh2ZEKA2peXr3ctKQwxI3R6+nrfg==",
	       "Timestamp": 1632503907836381400
	   }
	*/
	GenesisBlock   *pb.Block `json:"genesis_block" validate:"required"`
	GroupId        string    `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	GroupName      string    `json:"group_name" validate:"required" example:"demo group"`
	OwnerPubkey    string    `json:"owner_pubkey" validate:"required" example:"CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg"`
	ConsensusType  string    `json:"consensus_type" validate:"required,oneof=pos poa" example:"poa"`
	EncryptionType string    `json:"encryption_type" validate:"required,oneof=public private" example:"public"`
	CipherKey      string    `json:"cipher_key" validate:"required" example:"8e9bd83f84cf1408484d24f486861947a1db3fbe6eb3c61e31af55a4803aedc1"`
	AppKey         string    `json:"app_key" validate:"required" example:"test_app"`
	Signature      string    `json:"signature" validate:"required" example:"304502206897c3c67247cba2e8d5991501b3fd471fcca06f15915efdcd814b9e99c9a48a022100aa3024eb5663da6cbbde150132a4ff52c6c6aeeb49e0c039b4c28e72b071382f"`
}

type CreateGroupResult struct {
	Seed    string `json:"seed" validate:"required" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"` // seed url
	GroupId string `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
}

type GetGroupSeedResult struct {
	Seed string `json:"seed" validate:"required" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"` // seed url
}

func CreateGroup(params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*GroupSeed, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	if params.ConsensusType != "poa" {
		return nil, errors.New("consensus_type must be poa, other types are not supported yet")
	}

	//create groupid
	groupid := guuid.New().String()

	//init keystore
	ks := nodectx.GetNodeCtx().Keystore

	//init sign key
	ownerpubkey, err := initSignKey(groupid, ks, nodeoptions)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	//init cipher key
	cipherKey, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}

	//init encode key
	groupEncryptPubkey, err := initEncryptKey(groupid, ks)
	if err != nil {
		return nil, err
	}

	//create group item
	var item *pb.GroupItem
	item = &pb.GroupItem{}
	item.GroupId = groupid
	item.GroupName = params.GroupName
	item.OwnerPubKey = ownerpubkey
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
	item.LastUpdate = time.Now().UnixNano()

	group := &chain.Group{}
	genesisBlock, err := group.NewGroup(item)
	if err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	groupmgr.Groups[group.Item.GroupId] = group

	//create result
	encodedCipherKey := hex.EncodeToString(cipherKey)

	createGrpResult := &GroupSeed{
		GenesisBlock:   genesisBlock,
		GroupId:        groupid,
		GroupName:      params.GroupName,
		OwnerPubkey:    item.OwnerPubKey,
		ConsensusType:  params.ConsensusType,
		EncryptionType: params.EncryptionType,
		CipherKey:      encodedCipherKey,
		AppKey:         params.AppKey,
	}

	//get seed hash
	seedByts, err := json.Marshal(createGrpResult)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(seedByts)

	var signature []byte
	signature, err = ks.EthSignByKeyName(genesisBlock.GroupId, hash)
	if err != nil {
		return nil, err
	}
	if len(signature) == 0 {
		return nil, errors.New("create signature on group seed failed")
	}

	//set signature
	createGrpResult.Signature = hex.EncodeToString(signature)

	// save group seed to appdata
	pbGroupSeed := ToPbGroupSeed(*createGrpResult)
	if err := appdb.SetGroupSeed(&pbGroupSeed); err != nil {
		return nil, err
	}

	return createGrpResult, err
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

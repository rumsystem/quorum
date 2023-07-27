package handlers

import (
	"encoding/hex"
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
	"google.golang.org/protobuf/proto"
)

const DEFAULT_EPOCH_DURATION = 1000 //ms
const INIT_ANNOUNCE_TRX_ID = "00000000-0000-0000-0000-000000000001"
const INIT_FORK_TRX_ID = "00000000-0000-0000-0000-000000000000"

type CreateGroupParam struct {
	GroupName       string `from:"group_name"      json:"group_name"      validate:"required,max=100,min=2" example:"demo group"`
	ConsensusType   string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa" example:"poa"`
	EncryptionType  string `from:"encryption_type" json:"encryption_type" validate:"required,oneof=public private" example:"public"`
	AppKey          string `from:"app_key"         json:"app_key"         validate:"required,max=20,min=4" example:"test_app"`
	IncludeChainUrl bool   `json:"include_chain_url" example:"true"`
	JoinGroup       bool   `json:"join_group" example:"true"`
}

type JoinGroupParamV2 struct {
	Seed string `json:"seed" validate:"required" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"` // seed url
}

type CreateGroupResult struct {
	Seed    string `json:"seed" validate:"required" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"` // seed url
	GroupId string `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
}

type GetGroupSeedResult struct {
	Seed string `json:"seed" validate:"required" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"` // seed url
}

func CreateGroup(params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*pb.GroupSeed, error) {
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

	//init owner sign key
	ownerpubkey, err := initSignKey(groupid, ks, nodeoptions)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	//init owner encrypt key
	encryptPubkey, err := initEncryptKey(groupid, ks)
	if err != nil {
		return nil, err
	}

	//init cipher key
	cipherKey, err := localcrypto.CreateAesKey()
	if err != nil {
		return nil, err
	}

	var encryptType pb.GroupEncryptType
	if params.EncryptionType == "public" {
		encryptType = pb.GroupEncryptType_PUBLIC
	} else {
		encryptType = pb.GroupEncryptType_PRIVATE
	}

	//create group item
	item := &pb.GroupItem{
		GroupId:           groupid,
		GroupName:         params.GroupName,
		OwnerPubKey:       ownerpubkey,
		UserSignPubkey:    ownerpubkey,
		UserEncryptPubkey: encryptPubkey,
		ConsenseType:      pb.GroupConsenseType_POA,
		CipherKey:         hex.EncodeToString(cipherKey),
		AppKey:            params.AppKey,
		EncryptType:       encryptType,
		LastUpdate:        time.Now().UnixNano(),
	}

	//create announce trx
	announceTrx, err := createAnnounceTrx(ks, item)
	if err != nil {
		return nil, err
	}

	//create fork trx
	forkTrx, err := createForkTrx(ks, item)
	if err != nil {
		return nil, err
	}

	//create genesis block
	genesisBlock, err := createGenesisBlock(ks, item, announceTrx, forkTrx)
	if err != nil {
		return nil, err
	}

	//add genesis block to group item
	item.GenesisBlock = genesisBlock

	//create group seed
	seed := &pb.GroupSeed{
		GroupItem: item,
		Hash:      nil,
		Sign:      nil,
	}

	//hash and sign seed
	seedByts, err := proto.Marshal(seed)
	if err != nil {
		return nil, err
	}
	seed.Hash = localcrypto.Hash(seedByts)
	sign, err := ks.EthSignByKeyName(genesisBlock.GroupId, seed.Hash)
	if err != nil {
		return nil, err
	}
	seed.Sign = sign

	//check if join the group just created
	if !params.JoinGroup {
		return seed, nil
	}

	group := &chain.Group{}
	err = group.JoinGroup(item)
	if err != nil {
		return nil, err
	}

	if err := appdb.SetGroupSeed(seed); err != nil {
		return nil, err
	}

	return seed, err
}

/*
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
*/

func createForkTrx(ks localcrypto.Keystore, item *pb.GroupItem) (*pb.Trx, error) {
	/*

		//create initial consensus for genesis block
		consensusInfo := &pb.ConsensusInfo{
			ConsensusId:   uuid.New().String(),
			ChainVer:      0,
			InTrx:         INIT_FORK_TRX_ID,
			ForkFromBlock: 0,
		}

		//create fork info for genesis block
		forkItem := &pb.ForkItem{
			GroupId:        item.GroupId,
			Consensus:      consensusInfo,
			StartFromBlock: 0,
			StartFromEpoch: 0,
			EpochDuration:  DEFAULT_EPOCH_DURATION,
			Producers:      []string{item.OwnerPubKey}, //owner is the first producer
			Memo:           "genesis fork",
		}
	*/

	return nil, nil
}

func createAnnounceTrx(ks localcrypto.Keystore, item *pb.GroupItem) (*pb.Trx, error) {

	/*
		//owner announce as the first group producer
		group_log.Debugf("<%s> owner announce as the first group producer", grp.Item.GroupId)
		aContent := &quorumpb.AnnounceContent{
			Type:          quorumpb.AnnounceType_AS_PRODUCER,
			SignPubkey:    item.OwnerPubKey,
			EncryptPubkey: item.UserEncryptPubkey,
			Memo:          "owner announce as the first group producer",
		}

		aItem := &quorumpb.AnnounceItem{
			GroupId:         item.GroupId,
			Action:          quorumpb.ActionType_ADD,
			Content:         aContent,
			AnnouncerPubkey: item.OwnerPubKey,
		}

		//create hash
		byts, err := proto.Marshal(aItem)
		if err != nil {
			return nil, err
		}
		aItem.Hash = localcrypto.Hash(byts)
		signature, err := ks.EthSignByKeyName(item.GroupId, aItem.Hash)
		if err != nil {
			return nil, err
		}

		aItem.Signature = signature
	*/
	return nil, nil
}

func createGenesisBlock(ks localcrypto.Keystore, item *pb.GroupItem, announceTrx, forkTrx *pb.Trx) (*pb.Block, error) {
	return nil, nil
}

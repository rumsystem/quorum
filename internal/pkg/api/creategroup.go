package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type CreateGroupParam struct {
	GroupName      string `from:"group_name"      json:"group_name"      validate:"required,max=20,min=5"`
	ConsensusType  string `from:"consensus_type"  json:"consensus_type"  validate:"required,oneof=pos poa"`
	EncryptionType string `from:"encryption_type" json:"encryption_type" validate:"required,oneof=public private"`
	AppKey         string `from:"app_key"         json:"app_key"         validate:"required,max=20,min=5"`
}

type GroupSeed struct {
	GenesisBlock   *quorumpb.Block `json:"genesis_block" validate:"required"`
	GroupId        string          `json:"group_id" validate:"required"`
	GroupName      string          `json:"group_name" validate:"required,max=20,min=5"`
	OwnerPubkey    string          `json:"owner_pubkey" validate:"required"`
	ConsensusType  string          `json:"consensus_type" validate:"required,oneof=pos poa"`
	EncryptionType string          `json:"encryption_type" validate:"required,oneof=public private"`
	CipherKey      string          `json:"cipher_key" validate:"required"`
	AppKey         string          `json:"app_key" validate:"required"`
	Signature      string          `json:"signature" validate:"required"`
}

// @Tags Groups
// @Summary CreateGroup
// @Description Create a new group
// @Accept json
// @Produce json
// @Param data body CreateGroupParam true "GroupInfo"
// @Success 200 {object} GroupSeed
// @Router /api/v1/group [post]
func (h *Handler) CreateGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		validate := validator.New()
		params := new(CreateGroupParam)
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if params.ConsensusType != "poa" {
			output := "Other types of groups are not supported yet"
			return c.JSON(http.StatusBadRequest, output)
		}

		groupid := guuid.New()

		nodeoptions := options.GetNodeOptions()

		var groupSignPubkey []byte
		var p2ppubkey p2pcrypto.PubKey
		ks := nodectx.GetNodeCtx().Keystore
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if ok == true {
			hexkey, err := dirks.GetEncodedPubkey(groupid.String(), localcrypto.Sign)
			if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
				newsignaddr, err := dirks.NewKeyWithDefaultPassword(groupid.String(), localcrypto.Sign)
				if err == nil && newsignaddr != "" {
					_, err = dirks.NewKeyWithDefaultPassword(groupid.String(), localcrypto.Encrypt)
					if err == nil {
						err = nodeoptions.SetSignKeyMap(groupid.String(), newsignaddr)
						if err != nil {
							output[ERROR_INFO] = fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error())
							return c.JSON(http.StatusBadRequest, output)
						}
						hexkey, err = dirks.GetEncodedPubkey(groupid.String(), localcrypto.Sign)
					} else {
						output[ERROR_INFO] = "create new group key err:" + err.Error()
						return c.JSON(http.StatusBadRequest, output)
					}
				} else {
					output[ERROR_INFO] = "create new group key err:" + err.Error()
					return c.JSON(http.StatusBadRequest, output)
				}
			}
			pubkeybytes, err := hex.DecodeString(hexkey)
			p2ppubkey, err = p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
			groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
			if err != nil {
				output[ERROR_INFO] = "group key can't be decoded, err:" + err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		} else {
			output[ERROR_INFO] = fmt.Sprintf("unknown keystore type  %v:", ks)
			return c.JSON(http.StatusBadRequest, output)
		}

		genesisBlock, err := chain.CreateGenesisBlock(groupid.String(), p2ppubkey)

		if err != nil {
			output[ERROR_INFO] = "Create genesis block failed with msg:" + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		cipherKey, err := localcrypto.CreateAesKey()
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		groupEncryptPubkey, err := dirks.GetEncodedPubkey(groupid.String(), localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				groupEncryptPubkey, err = dirks.NewKeyWithDefaultPassword(groupid.String(), localcrypto.Encrypt)
				if err != nil {
					output[ERROR_INFO] = "Create key pair failed with msg:" + err.Error()
					return c.JSON(http.StatusBadRequest, output)
				}
			} else {
				output[ERROR_INFO] = "Create key pair failed with msg:" + err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		}

		//create group item
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
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusInternalServerError, output)
		}

		// save group seed to appdata
		pbGroupSeed := ToPbGroupSeed(*createGrpResult)
		if err := h.Appdb.SetGroupSeed(&pbGroupSeed); err != nil {
			output[ERROR_INFO] = fmt.Sprintf("save group seed failed: %d", err)
			return c.JSON(http.StatusInternalServerError, output)
		}

		// appdata.SetGroupSeed(groupid.String(), createGrpResult)
		return c.JSON(http.StatusOK, createGrpResult)
	}
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

// ToPbGroupSeed convert `api.GroupSeed` to `quorumpb.GroupSeed`
func ToPbGroupSeed(s GroupSeed) quorumpb.GroupSeed {
	return quorumpb.GroupSeed{
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

// FromPbGroupSeed convert `quorumpb.GroupSeed` to `api.GroupSeed`
func FromPbGroupSeed(s *quorumpb.GroupSeed) GroupSeed {
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

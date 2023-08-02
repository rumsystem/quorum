package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type JoinGroupBySeedParam struct {
	Seed []byte `from:"seed" json:"seed" validate:"required"`
}
type JoinGroupBySeedResult struct {
	AppId             string `json:"app_key" validate:"required" example:"test_app"`
	GroupId           string `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	GroupName         string `json:"group_name" validate:"required" example:"demo group"`
	OwnerPubkey       string `json:"owner_pubkey" validate:"required" example:"CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg=="`
	UserPubkey        string `json:"user_pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
	UserEncryptPubkey string `json:"user_encryptpubkey" validate:"required" example:"age1774tul0j5wy5y39saeg6enyst4gru2dwp7sjwgd4w9ahl6fkusxq3f8dcm"`
	ConsensusType     string `json:"consensus_type" validate:"required" example:"poa"`
	EncryptionType    string `json:"encryption_type" validate:"required" example:"public"`
	CipherKey         string `json:"cipher_key" validate:"required" example:"076a3cee50f3951744fbe6d973a853171139689fb48554b89f7765c0c6cbf15a"`
}

// @Tags Groups
// @Summary JoinGroupBySeed
// @Description Join a group by using group seed
// @Accept json
// @Produce json
// @Param data body handlers.JoinGroupBySeedParam true "JoinGroupBySeedParam"
// @Success 200 {object} JoinGroupBySeedResult
// @Router /api/v2/group/join [post]
func (h *Handler) JoinGroupBySeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		payload := new(JoinGroupBySeedParam)
		if err := cc.BindAndValidate(payload); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//unmarshal seed
		seed := &quorumpb.GroupSeedRumLite{}
		err := proto.Unmarshal(payload.Seed, seed)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		groupItem := seed.Group

		//check if group exist
		groupmgr := chain.GetGroupMgr()
		if _, ok := groupmgr.Groups[groupItem.GroupId]; ok {
			msg := fmt.Sprintf("group with group_id <%s> already exist", groupItem.GroupId)
			return rumerrors.NewBadRequestError(msg)
		}

		//verify hash and signature
		hash := localcrypto.Hash(payload.Seed)
		if bytes.Compare(hash, seed.Hash) != 0 {
			msg := fmt.Sprintf("hash not match, expect %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(seed.Hash))
			return rumerrors.NewBadRequestError(msg)
		}

		verified, err := rumchaindata.VerifySign(groupItem.OwnerPubKey, seed.Signature, hash)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if !verified {
			msg := fmt.Sprintf("verify signature failed")
			return rumerrors.NewBadRequestError(msg)
		}

		//verify genesis block
		r, err := rumchaindata.ValidGenesisBlockRumLite(groupItem.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if !r {
			msg := "Join Group failed, verify genesis block failed"
			return rumerrors.NewBadRequestError(msg)
		}

		//create empty group
		group := &chain.GroupRumLite{}
		err = group.JoinGroup(groupItem)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		/*
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			genesisBlockBytes, err := json.Marshal(seed.GenesisBlock)
			if err != nil {
				msg := fmt.Sprintf("unmarshal genesis block failed with msg: %s" + err.Error())
				return rumerrors.NewBadRequestError(msg)
			}

			//TBD check if group already exist
			groupmgr := chain.GetGroupMgr()
			if _, ok := groupmgr.Groups[seed.GroupId]; ok {
				msg := fmt.Sprintf("group with group_id <%s> already exist", seed.GroupId)
				return rumerrors.NewBadRequestError(msg)
			}

			nodeoptions := options.GetNodeOptions()

			var groupSignPubkey []byte
			ks := nodectx.GetNodeCtx().Keystore
			dirks, ok := ks.(*localcrypto.DirKeyStore)
			if ok {
				base64key, err := dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Sign)
				if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
					newsignaddr, err := dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Sign)
					if err == nil && newsignaddr != "" {
						_, _ = dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
						err = nodeoptions.SetSignKeyMap(seed.GenesisBlock.GroupId, newsignaddr)
						if err != nil {
							msg := fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error())
							return rumerrors.NewBadRequestError(msg)
						}
						base64key, _ = dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Sign)
					} else {
						_, err := dirks.GetKeyFromUnlocked(localcrypto.Sign.NameString(seed.GenesisBlock.GroupId))
						if err != nil {
							msg := "create new group key err:" + err.Error()
							return rumerrors.NewBadRequestError(msg)
						}
						base64key, _ = dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Sign)
					}
				}
				groupSignPubkey, err = base64.RawURLEncoding.DecodeString(base64key)
				if err != nil {
					msg := "group key can't be decoded, err:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
			} else {
				msg := fmt.Sprintf("unknown keystore type  %v:", ks)
				return rumerrors.NewBadRequestError(msg)
			}

			ownerPubkeyBytes, err := base64.RawURLEncoding.DecodeString(seed.GenesisBlock.ProducerPubkey)
			if err != nil {
				msg := "Decode OwnerPubkey failed: " + err.Error()
				return rumerrors.NewBadRequestError(msg)
			}

			groupEncryptkey, err := dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
			if err != nil {
				if strings.HasPrefix(err.Error(), "key not exist") {
					_, _ = dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
					_, err := dirks.GetKeyFromUnlocked(localcrypto.Encrypt.NameString(seed.GenesisBlock.GroupId))
					if err != nil {
						msg := "Create key pair failed with msg:" + err.Error()
						return rumerrors.NewBadRequestError(msg)
					}
					groupEncryptkey, _ = dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
				} else {
					msg := "Create key pair failed with msg:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
			}

			r, err := rumchaindata.ValidGenesisBlock(seed.GenesisBlock)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			if !r {
				msg := "Join Group failed, verify genesis block failed"
				return rumerrors.NewBadRequestError(msg)
			}

			item := &quorumpb.GroupItem{}
			item.GroupId = seed.GroupId
			item.GroupName = seed.GroupName
			item.OwnerPubKey = seed.OwnerPubkey

			item.CipherKey = seed.CipherKey
			item.AppKey = seed.AppKey

			if seed.ConsensusType == "poa" {
				item.ConsenseType = quorumpb.GroupConsenseType_POA
			} else if seed.ConsensusType == "pos" {
				item.ConsenseType = quorumpb.GroupConsenseType_POS
			}

			item.UserSignPubkey = base64.RawURLEncoding.EncodeToString(groupSignPubkey)

			userEncryptKey, err := dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
			if err != nil {
				if strings.HasPrefix(err.Error(), "key not exist") {
					userEncryptKey, err = dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
					if err != nil {
						msg := "Create key pair failed with msg:" + err.Error()
						return rumerrors.NewBadRequestError(msg)
					}
				} else {
					msg := "Create key pair failed with msg:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
			}

			item.UserEncryptPubkey = userEncryptKey
			if seed.EncryptionType == "public" {
				item.EncryptType = quorumpb.GroupEncryptType_PUBLIC
			} else {
				item.EncryptType = quorumpb.GroupEncryptType_PRIVATE
			}

			item.LastUpdate = seed.GenesisBlock.TimeStamp
			item.GenesisBlock = seed.GenesisBlock

			//create the group
			group := &chain.Group{}
			err = group.JoinGroup(item)

			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			//start sync
			err = group.StartSync()
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			//add group to context
			groupmgr.Groups[group.Item.GroupId] = group

			var bufferResult bytes.Buffer
			bufferResult.Write(genesisBlockBytes)
			bufferResult.Write([]byte(item.GroupId))
			bufferResult.Write([]byte(item.GroupName))
			bufferResult.Write(ownerPubkeyBytes)
			bufferResult.Write(groupSignPubkey)
			bufferResult.Write([]byte(groupEncryptkey))
			bufferResult.Write([]byte(item.CipherKey))
			hashResult := localcrypto.Hash(bufferResult.Bytes())
			signature, _ := ks.EthSignByKeyName(item.GroupId, hashResult)
			encodedSign := hex.EncodeToString(signature)

			joinGrpResult := &JoinGroupResult{
				GroupId:           item.GroupId,
				GroupName:         item.GroupName,
				OwnerPubkey:       item.OwnerPubKey,
				ConsensusType:     seed.ConsensusType,
				EncryptionType:    seed.EncryptionType,
				UserPubkey:        item.UserSignPubkey,
				UserEncryptPubkey: groupEncryptkey,
				CipherKey:         item.CipherKey,
				AppKey:            item.AppKey,
				Signature:         encodedSign,
			}

			// save group seed to appdata
			pbGroupSeed := handlers.ToPbGroupSeed(*seed)
			if err := h.Appdb.SetGroupSeed(&pbGroupSeed); err != nil {
				msg := fmt.Sprintf("save group seed failed: %s", err)
				return rumerrors.NewBadRequestError(msg)
			}

			return c.JSON(http.StatusOK, joinGrpResult)
		*/

		return c.JSON(http.StatusOK, nil)
	}
}

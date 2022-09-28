package api

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
)

type JoinGroupResult struct {
	GroupId           string `json:"group_id" validate:"required"`
	GroupName         string `json:"group_name" validate:"required"`
	OwnerPubkey       string `json:"owner_pubkey" validate:"required"`
	UserPubkey        string `json:"user_pubkey" validate:"required"`
	UserEncryptPubkey string `json:"user_encryptpubkey" validate:"required"`
	ConsensusType     string `json:"consensus_type" validate:"required"`
	EncryptionType    string `json:"encryption_type" validate:"required"`
	CipherKey         string `json:"cipher_key" validate:"required"`
	AppKey            string `json:"app_key" validate:"required"`
	Signature         string `json:"signature" validate:"required"`
}

func (h *Handler) JoinGroupV2() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		payload := new(handlers.JoinGroupParamV2)
		if err := cc.BindAndValidate(payload); err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		seed, _, err := handlers.UrlToGroupSeed(payload.Seed)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		genesisBlockBytes, err := json.Marshal(seed.GenesisBlock)
		if err != nil {
			msg := fmt.Sprintf("unmarshal genesis block failed with msg: %s" + err.Error())
			return rumerrors.NewBadRequestError(msg)
		}

		nodeoptions := options.GetNodeOptions()

		var groupSignPubkey []byte
		ks := nodectx.GetNodeCtx().Keystore
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if ok == true {
			base64key, err := dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Sign)
			if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
				newsignaddr, err := dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Sign)
				if err == nil && newsignaddr != "" {
					_, err = dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
					err = nodeoptions.SetSignKeyMap(seed.GenesisBlock.GroupId, newsignaddr)
					if err != nil {
						msg := fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error())
						return rumerrors.NewBadRequestError(msg)
					}
					base64key, err = dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Sign)
				} else {
					_, err := dirks.GetKeyFromUnlocked(localcrypto.Sign.NameString(seed.GenesisBlock.GroupId))
					if err != nil {
						msg := "create new group key err:" + err.Error()
						return rumerrors.NewBadRequestError(msg)
					}
					base64key, err = dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Sign)
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

		ownerPubkeyBytes, err := base64.RawURLEncoding.DecodeString(seed.GenesisBlock.ProducerPubKey)
		if err != nil {
			//the key maybe a libp2p key, try...
			ownerPubkeyBytes, err = p2pcrypto.ConfigDecodeKey(seed.GenesisBlock.ProducerPubKey)
			if err != nil {
				msg := "Decode OwnerPubkey failed: " + err.Error()
				return rumerrors.NewBadRequestError(msg)
			}
		}

		groupEncryptkey, err := dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				groupEncryptkey, err = dirks.NewKeyWithDefaultPassword(seed.GenesisBlock.GroupId, localcrypto.Encrypt)

				_, err := dirks.GetKeyFromUnlocked(localcrypto.Encrypt.NameString(seed.GenesisBlock.GroupId))
				if err != nil {
					msg := "Create key pair failed with msg:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
				groupEncryptkey, err = dirks.GetEncodedPubkey(seed.GenesisBlock.GroupId, localcrypto.Encrypt)
			} else {
				msg := "Create key pair failed with msg:" + err.Error()
				return rumerrors.NewBadRequestError(msg)
			}
		}

		r, err := rumchaindata.VerifyBlockSign(seed.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if r == false {
			msg := "Join Group failed, can not verify signature"
			return rumerrors.NewBadRequestError(msg)
		}

		var item *quorumpb.GroupItem
		item = &quorumpb.GroupItem{}

		item.OwnerPubKey = seed.GenesisBlock.ProducerPubKey
		item.GroupId = seed.GenesisBlock.GroupId
		item.GroupName = seed.GroupName
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

		item.HighestBlockId = seed.GenesisBlock.BlockId
		item.HighestHeight = 0
		item.LastUpdate = seed.GenesisBlock.TimeStamp
		item.GenesisBlock = seed.GenesisBlock

		//create the group
		var group *chain.Group
		group = &chain.Group{}
		err = group.CreateGrp(item)
		if nodeoptions.IsRexTestMode == true {
			group.SetRumExchangeTestMode()
		}
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//start sync
		err = group.StartSync(false)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//add group to context
		groupmgr := chain.GetGroupMgr()
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
		signature, err := ks.EthSignByKeyName(item.GroupId, hashResult)
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
	}
}

// JoinGroupByHTTPRequest restore cli use it
func JoinGroupByHTTPRequest(apiBaseUrl string, payload *handlers.CreateGroupResult) (*JoinGroupResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		e := fmt.Errorf("json.Marshal failed: %s, joinGroupParam: %+v", err, payload)
		return nil, e
	}

	payloadStr := string(payloadByte[:])
	urlPath := "/api/v2/group/join"
	_, resp, err := testnode.RequestAPI(apiBaseUrl, urlPath, "POST", payloadStr)
	if err != nil {
		e := fmt.Errorf("request %s failed: %s, payload: %s", urlPath, err, payloadStr)
		return nil, e
	}

	var result JoinGroupResult
	if err := json.Unmarshal(resp, &result); err != nil {
		e := fmt.Errorf("json.Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	return &result, nil
}

package api

import (
	//"encoding/json"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/testnode"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
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

// @Tags Groups
// @Summary JoinGroup
// @Description Join a group
// @Accept json
// @Produce json
// @Param data body handlers.GroupSeed true "GroupSeed"
// @Success 200 {object} JoinGroupResult
// @Router /api/v1/group/join [post]
func (h *Handler) JoinGroup() echo.HandlerFunc {
	return func(c echo.Context) error {

		var err error
		output := make(map[string]string)
		validate := validator.New()
		params := new(handlers.GroupSeed)

		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		genesisBlockBytes, err := json.Marshal(params.GenesisBlock)
		if err != nil {
			output[ERROR_INFO] = "unmarshal genesis block failed with msg:" + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		nodeoptions := options.GetNodeOptions()

		var groupSignPubkey []byte
		ks := nodectx.GetNodeCtx().Keystore
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if ok == true {
			hexkey, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
			if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
				newsignaddr, err := dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Sign)
				if err == nil && newsignaddr != "" {
					_, err = dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Encrypt)
					err = nodeoptions.SetSignKeyMap(params.GroupId, newsignaddr)
					if err != nil {
						output[ERROR_INFO] = fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error())
						return c.JSON(http.StatusBadRequest, output)
					}
					hexkey, err = dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
				} else {
					_, err := dirks.GetKeyFromUnlocked(localcrypto.Sign.NameString(params.GroupId))
					if err != nil {
						output[ERROR_INFO] = "create new group key err:" + err.Error()
						return c.JSON(http.StatusBadRequest, output)
					}
					hexkey, err = dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
				}
			}

			pubkeybytes, err := hex.DecodeString(hexkey)
			p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
			groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
			if err != nil {
				output[ERROR_INFO] = "group key can't be decoded, err:" + err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		} else {
			output[ERROR_INFO] = fmt.Sprintf("unknown keystore type  %v:", ks)
			return c.JSON(http.StatusBadRequest, output)
		}

		ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.OwnerPubkey)
		if err != nil {
			output[ERROR_INFO] = "Decode OwnerPubkey failed " + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		ownerPubkey, err := p2pcrypto.UnmarshalPublicKey(ownerPubkeyBytes)
		if err != nil {
			return c.JSON(http.StatusBadRequest, output)
		}

		//decode signature
		decodedSignature, err := hex.DecodeString(params.Signature)
		if err != nil {
			return c.JSON(http.StatusBadRequest, output)
		}

		//decode cipherkey
		cipherKey, err := hex.DecodeString(params.CipherKey)
		if err != nil {
			return c.JSON(http.StatusBadRequest, output)
		}

		groupEncryptkey, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				groupEncryptkey, err = dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Encrypt)

				_, err := dirks.GetKeyFromUnlocked(localcrypto.Encrypt.NameString(params.GroupId))
				if err != nil {
					output[ERROR_INFO] = "Create key pair failed with msg:" + err.Error()
					return c.JSON(http.StatusBadRequest, output)
				}
				groupEncryptkey, err = dirks.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
			} else {
				output[ERROR_INFO] = "Create key pair failed with msg:" + err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		}

		var buffer bytes.Buffer
		buffer.Write(genesisBlockBytes)
		buffer.Write([]byte(params.GroupId))
		buffer.Write([]byte(params.GroupName))
		buffer.Write(ownerPubkeyBytes)
		buffer.Write([]byte(params.ConsensusType))
		buffer.Write([]byte(params.EncryptionType))
		buffer.Write([]byte(params.AppKey))
		buffer.Write(cipherKey)

		hash := localcrypto.Hash(buffer.Bytes())
		verifiy, err := ownerPubkey.Verify(hash, decodedSignature)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if !verifiy {
			output[ERROR_INFO] = "Join Group failed, can not verify signature"
			return c.JSON(http.StatusBadRequest, output)
		}

		var item *quorumpb.GroupItem
		item = &quorumpb.GroupItem{}

		item.OwnerPubKey = params.OwnerPubkey
		item.GroupId = params.GroupId
		item.GroupName = params.GroupName
		item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(ownerPubkeyBytes)
		item.CipherKey = params.CipherKey
		item.AppKey = params.AppKey

		item.ConsenseType = quorumpb.GroupConsenseType_POA
		item.UserSignPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)

		userEncryptKey, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				userEncryptKey, err = dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Encrypt)
				if err != nil {
					output[ERROR_INFO] = "Create key pair failed with msg:" + err.Error()
					return c.JSON(http.StatusBadRequest, output)
				}
			} else {
				output[ERROR_INFO] = "Create key pair failed with msg:" + err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		}

		item.UserEncryptPubkey = userEncryptKey
		item.UserSignPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)

		if params.EncryptionType == "public" {
			item.EncryptType = quorumpb.GroupEncryptType_PUBLIC
		} else {
			item.EncryptType = quorumpb.GroupEncryptType_PRIVATE
		}

		item.HighestBlockId = params.GenesisBlock.BlockId
		item.HighestHeight = 0
		item.LastUpdate = time.Now().UnixNano()
		item.GenesisBlock = params.GenesisBlock

		//create the group
		var group *chain.Group
		group = &chain.Group{}
		err = group.CreateGrp(item)
		if nodeoptions.IsRexTestMode == true {
			group.SetRumExchangeTestMode()
		}
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		//start sync
		err = group.StartSync()
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
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
		buffer.Write([]byte(params.ConsensusType))
		buffer.Write([]byte(params.EncryptionType))
		buffer.Write([]byte(item.CipherKey))
		buffer.Write([]byte(item.AppKey))
		hashResult := chain.Hash(bufferResult.Bytes())
		signature, err := ks.SignByKeyName(item.GroupId, hashResult)
		encodedSign := hex.EncodeToString(signature)

		joinGrpResult := &JoinGroupResult{GroupId: item.GroupId, GroupName: item.GroupName, OwnerPubkey: item.OwnerPubKey, ConsensusType: params.ConsensusType, EncryptionType: params.EncryptionType, UserPubkey: item.UserSignPubkey, UserEncryptPubkey: groupEncryptkey, CipherKey: item.CipherKey, AppKey: item.AppKey, Signature: encodedSign}

		// save group seed to appdata
		pbGroupSeed := handlers.ToPbGroupSeed(*params)
		if err := h.Appdb.SetGroupSeed(&pbGroupSeed); err != nil {
			output[ERROR_INFO] = fmt.Sprintf("save group seed failed: %s", err)
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, joinGrpResult)
	}
}

// JoinGroupByHTTPRequest restore cli use it
func JoinGroupByHTTPRequest(apiBaseUrl string, payload handlers.GroupSeed) (*JoinGroupResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		e := fmt.Errorf("json.Marshal failed: %s, joinGroupParam: %+v", err, payload)
		return nil, e
	}

	payloadStr := string(payloadByte[:])
	urlPath := "/api/v1/group/join"
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

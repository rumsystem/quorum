package api

import (
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

type JoinGroupResult struct {
	GroupId       string `json:"group_id" validate:"required,uuid4" example:"c0020941-e648-40c9-92dc-682645acd17e"`
	GroupName     string `json:"group_name" validate:"required" example:"demo group"`
	OwnerPubkey   string `json:"owner_pubkey" validate:"required" example:"CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg=="`
	UserPubkey    string `json:"user_pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
	ConsensusType string `json:"consensus_type" validate:"required" example:"poa"`
	SyncType      string `json:"sync_type" validate:"required" example:"public"`
	CipherKey     string `json:"cipher_key" validate:"required" example:"076a3cee50f3951744fbe6d973a853171139689fb48554b89f7765c0c6cbf15a"`
	AppKey        string `json:"app_key" validate:"required" example:"test_app"`
	Signature     string `json:"signature" validate:"required" example:"3045022100a819a627237e0bb0de1e69e3b29119efbf8677173f7e4d3a20830fc366c5bfd702200ad71e34b53da3ac5bcf3f8a46f1964b058ef36c2687d3b8effe4baec2acd2a6"`
}

// @Tags Groups
// @Summary JoinGroup
// @Description Join a group by using group seed
// @Accept json
// @Produce json
// @Param data body handlers.JoinGroupParamV2 true "JoinGroupParamV2"
// @Success 200 {object} JoinGroupResult
// @Router /api/v2/group/join [post]
func (h *Handler) JoinGroupV2() echo.HandlerFunc {
	return func(c echo.Context) error {
		//Commented by cuicat
		/*
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

			if seed.SyncType == "public" {
				item.SyncType = quorumpb.GroupSyncType_PUBLIC
			} else {
				item.SyncType = quorumpb.GroupSyncType_PRIVATE
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
				GroupId:       item.GroupId,
				GroupName:     item.GroupName,
				OwnerPubkey:   item.OwnerPubKey,
				ConsensusType: seed.ConsensusType,
				SyncType:      seed.SyncType,
				UserPubkey:    item.UserSignPubkey,
				CipherKey:     item.CipherKey,
				AppKey:        item.AppKey,
				Signature:     encodedSign,
			}

			// save group seed to appdata
			pbGroupSeed := handlers.ToPbGroupSeed(*seed)
			if err := h.Appdb.SetGroupSeed(&pbGroupSeed); err != nil {
				msg := fmt.Sprintf("save group seed failed: %s", err)
				return rumerrors.NewBadRequestError(msg)
			}

			return c.JSON(http.StatusOK, joinGrpResult)
		*/

		return nil
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

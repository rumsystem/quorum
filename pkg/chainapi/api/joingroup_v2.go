package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"github.com/rumsystem/quorum/testnode"
	"google.golang.org/protobuf/proto"

	rumchaindata "github.com/rumsystem/quorum/pkg/data"
)

type JoinGroupResult struct {
	GroupItem *quorumpb.GroupItem `json:"group_item"`
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
		cc := c.(*utils.CustomContext)

		payload := new(handlers.JoinGroupParamV2)
		if err := cc.BindAndValidate(payload); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//get seed from url
		seed, _, err := handlers.UrlToGroupSeed(payload.Seed)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//verify seed
		seedBytes, err := proto.Marshal(seed)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		hash := localcrypto.Hash(seedBytes)
		if !bytes.Equal(hash, seed.Hash) {
			return rumerrors.NewBadRequestError("invalid seed hash")
		}

		ks := nodectx.GetNodeCtx().Keystore
		bytespubkey, err := base64.RawURLEncoding.DecodeString(seed.GroupItem.OwnerPubKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		verified := ks.EthVerifySign(hash, seed.Sign, ethpubkey)
		if !verified {
			return rumerrors.NewBadRequestError("invalid seed signature")
		}

		//verify genesis block
		isGenesisBlockValid, err := rumchaindata.ValidGenesisBlock(seed.GroupItem.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if !isGenesisBlockValid {
			return rumerrors.NewBadRequestError("invalid genesis block")
		}

		groupId := seed.GroupItem.GroupId

		//TBD check if group already exist
		groupmgr := chain.GetGroupMgr()
		if _, ok := groupmgr.Groups[groupId]; ok {
			msg := fmt.Sprintf("group with group_id <%s> already exist", groupId)
			return rumerrors.NewBadRequestError(msg)
		}

		nodeoptions := options.GetNodeOptions()

		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			msg := fmt.Sprintf("unknown keystore type  %v:", ks)
			return rumerrors.NewBadRequestError(msg)
		}

		userSignPubkey, err := dirks.GetEncodedPubkey(groupId, localcrypto.Sign)
		if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
			newsignaddr, err := dirks.NewKeyWithDefaultPassword(groupId, localcrypto.Sign)
			if err == nil && newsignaddr != "" {
				_, _ = dirks.NewKeyWithDefaultPassword(groupId, localcrypto.Encrypt)
				err = nodeoptions.SetSignKeyMap(groupId, newsignaddr)
				if err != nil {
					msg := fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error())
					return rumerrors.NewBadRequestError(msg)
				}
				userSignPubkey, _ = dirks.GetEncodedPubkey(groupId, localcrypto.Sign)
			} else {
				_, err := dirks.GetKeyFromUnlocked(localcrypto.Sign.NameString(groupId))
				if err != nil {
					msg := "create new group key err:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
				userSignPubkey, _ = dirks.GetEncodedPubkey(groupId, localcrypto.Sign)
			}
		}

		userEncryptkey, err := dirks.GetEncodedPubkey(groupId, localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				_, _ = dirks.NewKeyWithDefaultPassword(groupId, localcrypto.Encrypt)
				_, err := dirks.GetKeyFromUnlocked(localcrypto.Encrypt.NameString(groupId))
				if err != nil {
					msg := "Create key pair failed with msg:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
				userEncryptkey, _ = dirks.GetEncodedPubkey(groupId, localcrypto.Encrypt)
			} else {
				msg := "Create key pair failed with msg:" + err.Error()
				return rumerrors.NewBadRequestError(msg)
			}
		}

		groupItem := &quorumpb.GroupItem{
			GroupId:           seed.GroupItem.GroupId,
			GroupName:         seed.GroupItem.GroupName,
			OwnerPubKey:       seed.GroupItem.OwnerPubKey,
			UserSignPubkey:    userSignPubkey,
			UserEncryptPubkey: userEncryptkey,
			ConsenseType:      seed.GroupItem.ConsenseType,
			CipherKey:         seed.GroupItem.CipherKey,
			AppKey:            seed.GroupItem.AppKey,
			EncryptType:       seed.GroupItem.EncryptType,
			LastUpdate:        seed.GroupItem.LastUpdate,
			GenesisBlock:      seed.GroupItem.GenesisBlock,
		}

		//create the group
		group := &chain.Group{}
		err = group.JoinGroup(groupItem)

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

		// save group seed to appdata
		if err := h.Appdb.SetGroupSeed(seed); err != nil {
			msg := fmt.Sprintf("save group seed failed: %s", err)
			return rumerrors.NewBadRequestError(msg)
		}

		joinGrpResult := &JoinGroupResult{
			GroupItem: groupItem,
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

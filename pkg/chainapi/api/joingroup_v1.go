package api

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
)

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
		cc := c.(*utils.CustomContext)

		params := new(handlers.GroupSeed)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		genesisBlockBytes, err := json.Marshal(params.GenesisBlock)
		if err != nil {
			msg := fmt.Sprintf("unmarshal genesis block failed with msg: %s" + err.Error())
			return rumerrors.NewBadRequestError(msg)
		}

		nodeoptions := options.GetNodeOptions()

		var groupSignPubkey []byte
		ks := nodectx.GetNodeCtx().Keystore
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if ok == true {
			base64key, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
			if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
				newsignaddr, err := dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Sign)
				if err == nil && newsignaddr != "" {
					_, err = dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Encrypt)
					err = nodeoptions.SetSignKeyMap(params.GroupId, newsignaddr)
					if err != nil {
						msg := fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error())
						return rumerrors.NewBadRequestError(msg)
					}
					base64key, err = dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
				} else {
					_, err := dirks.GetKeyFromUnlocked(localcrypto.Sign.NameString(params.GroupId))
					if err != nil {
						msg := "create new group key err:" + err.Error()
						return rumerrors.NewBadRequestError(msg)
					}
					base64key, err = dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
				}
			}
			groupSignPubkey, err = base64.RawURLEncoding.DecodeString(base64key)
			if err != nil {
				msg := "group key can't be decoded, err: " + err.Error()
				return rumerrors.NewBadRequestError(msg)
			}
		} else {
			msg := fmt.Sprintf("unknown keystore type  %v:", ks)
			return rumerrors.NewBadRequestError(msg)
		}

		ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.OwnerPubkey)
		if err != nil {
			msg := "Decode OwnerPubkey failed: " + err.Error()
			return rumerrors.NewBadRequestError(msg)
		}

		ownerPubkey, err := p2pcrypto.UnmarshalPublicKey(ownerPubkeyBytes)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//decode signature
		decodedSignature, err := hex.DecodeString(params.Signature)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//decode cipherkey
		cipherKey, err := hex.DecodeString(params.CipherKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		groupEncryptkey, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				groupEncryptkey, err = dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Encrypt)

				_, err := dirks.GetKeyFromUnlocked(localcrypto.Encrypt.NameString(params.GroupId))
				if err != nil {
					msg := "Create key pair failed with msg:" + err.Error()
					return rumerrors.NewBadRequestError(msg)
				}
				groupEncryptkey, err = dirks.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
			} else {
				msg := "Create key pair failed with msg:" + err.Error()
				return rumerrors.NewBadRequestError(msg)
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
			return rumerrors.NewBadRequestError(err)
		}

		if !verifiy {
			return rumerrors.NewBadRequestError("Join Group failed, can not verify signature")
		}

		var item *quorumpb.GroupItem
		item = &quorumpb.GroupItem{}

		item.OwnerPubKey = params.OwnerPubkey
		item.GroupId = params.GroupId
		item.GroupName = params.GroupName

		secp256k1pubkey, ok := ownerPubkey.(*p2pcrypto.Secp256k1PublicKey)
		if ok == true {
			btcecpubkey := (*btcec.PublicKey)(secp256k1pubkey)
			item.OwnerPubKey = base64.RawURLEncoding.EncodeToString(ethcrypto.CompressPubkey(btcecpubkey.ToECDSA()))
		}

		item.CipherKey = params.CipherKey
		item.AppKey = params.AppKey

		item.ConsenseType = quorumpb.GroupConsenseType_POA
		item.UserSignPubkey = base64.RawURLEncoding.EncodeToString(groupSignPubkey)

		userEncryptKey, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
		if err != nil {
			if strings.HasPrefix(err.Error(), "key not exist") {
				userEncryptKey, err = dirks.NewKeyWithDefaultPassword(params.GroupId, localcrypto.Encrypt)
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
		//item.UserSignPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)

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
		buffer.Write([]byte(params.ConsensusType))
		buffer.Write([]byte(params.EncryptionType))
		buffer.Write([]byte(item.CipherKey))
		buffer.Write([]byte(item.AppKey))
		hashResult := localcrypto.Hash(bufferResult.Bytes())
		signature, err := ks.EthSignByKeyName(item.GroupId, hashResult)
		encodedSign := hex.EncodeToString(signature)

		joinGrpResult := &JoinGroupResult{GroupId: item.GroupId, GroupName: item.GroupName, OwnerPubkey: item.OwnerPubKey, ConsensusType: params.ConsensusType, EncryptionType: params.EncryptionType, UserPubkey: item.UserSignPubkey, UserEncryptPubkey: groupEncryptkey, CipherKey: item.CipherKey, AppKey: item.AppKey, Signature: encodedSign}

		// save group seed to appdata
		pbGroupSeed := handlers.ToPbGroupSeed(*params)
		if err := h.Appdb.SetGroupSeed(&pbGroupSeed); err != nil {
			msg := fmt.Sprintf("save group seed failed: %s", err)
			return rumerrors.NewBadRequestError(msg)
		}

		return c.JSON(http.StatusOK, joinGrpResult)
	}
}

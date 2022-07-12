package nodesdkapi

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type JoinGroupParams struct {
	Seed         handlers.GroupSeed `json:"seed"          validate:"required"`
	ChainAPIUrl  []string           `json:"urls"          validate:"required"`
	SignAlias    string             `json:"sign_alias"    validate:"required"`
	EncryptAlias string             `json:"encrypt_alias" validate:"required"`
}

type JoinGroupResult struct {
	GroupId        string `json:"group_id" validate:"required"`
	GroupName      string `json:"group_name" validate:"required"`
	OwnerPubkey    string `json:"owner_pubkey" validate:"required"`
	SignAlias      string `json:"sign_alias" validate:"required"`
	EncryptAlias   string `json:"encrypt_alias" validate:"required"`
	ConsensusType  string `json:"consensus_type" validate:"required"`
	EncryptionType string `json:"encryption_type" validate:"required"`
	CipherKey      string `json:"cipher_key" validate:"required"`
	AppKey         string `json:"app_key" validate:"required"`
	Signature      string `json:"signature" validate:"required"`
}

// isValidUrl tests a string to determine if it is a well-structured url or not.
func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

func (h *NodeSDKHandler) JoinGroupV2() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)
		params := make(map[string]string)

		if err = c.Bind(&params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		seed, serverurls, err := handlers.UrlToGroupSeed(params["seed"])
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
			cc := c.(*utils.CustomContext)

			params := new(JoinGroupParams)
			if err := cc.BindAndValidate(params); err != nil {
				return err
			}
			SignAlias := params["sign_alias"]
			EncryptAlias := params["encrypt_alias"]
			genesisBlockBytes, err := json.Marshal(seed.GenesisBlock)
			if err != nil {
				return rumerrors.NewBadRequestError("unmarshal genesis block failed with msg:" + err.Error())
			}

			ks := nodesdkctx.GetKeyStore()
			dirks, ok := ks.(*localcrypto.DirKeyStore)
			if !ok {
				return rumerrors.NewBadRequestError(rumerrors.ErrOpenKeystore)
			}

			signKeyName := dirks.AliasToKeyname(SignAlias)
			if signKeyName == "" {
				return rumerrors.NewBadRequestError(rumerrors.ErrSignAliasNotFound)
			}

			encryptKeyName := dirks.AliasToKeyname(EncryptAlias)
			if encryptKeyName == "" {
				return rumerrors.NewBadRequestError(rumerrors.ErrEncryptAliasNotFound)
			}

			// should check keytype
			allKeys, err := dirks.ListAll()
			if err != nil {
				return rumerrors.NewBadRequestError("ListAll failed")
			}

			//expand and dump all alias to a map
			var allAlias map[string]*localcrypto.KeyItem
			allAlias = make(map[string]*localcrypto.KeyItem)
			for _, item := range allKeys {
				for _, alias := range item.Alias {
					allAlias[alias] = item
				}
			}

			//check if given alias exist
			//check type
			alias, ok := allAlias[EncryptAlias]
			if !ok {
				return rumerrors.NewBadRequestError(rumerrors.ErrEncryptAliasNotFound)
			}

			if alias.Type != localcrypto.Encrypt {
				return rumerrors.NewBadRequestError(rumerrors.ErrInvalidAliasType)
			}

			alias, ok = allAlias[SignAlias]
			if !ok {
				return rumerrors.NewBadRequestError(rumerrors.ErrSignAliasNotFound)
			}

			if alias.Type != localcrypto.Sign {
				return rumerrors.NewBadRequestError(rumerrors.ErrInvalidAliasType)
			}

			ownerPubkeyBytes, err := base64.RawURLEncoding.DecodeString(seed.GenesisBlock.ProducerPubKey)
			r, err := rumchaindata.VerifyBlockSign(seed.GenesisBlock)
			//check seed signature
			ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.Seed.OwnerPubkey)
			if err != nil {
				return rumerrors.NewBadRequestError("Decode OwnerPubkey failed: " + err.Error())
			}

			ownerPubkey, err := p2pcrypto.UnmarshalPublicKey(ownerPubkeyBytes)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			//decode signature
			decodedSignature, err := hex.DecodeString(params.Seed.Signature)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			//decode cipherkey
			cipherKey, err := hex.DecodeString(params.Seed.CipherKey)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			var buffer bytes.Buffer
			buffer.Write(genesisBlockBytes)
			buffer.Write([]byte(params.Seed.GroupId))
			buffer.Write([]byte(params.Seed.GroupName))
			buffer.Write(ownerPubkeyBytes)
			buffer.Write([]byte(params.Seed.ConsensusType))
			buffer.Write([]byte(params.Seed.EncryptionType))
			buffer.Write([]byte(params.Seed.AppKey))
			buffer.Write(cipherKey)

			hash := localcrypto.Hash(buffer.Bytes())
			verifiy, err := ownerPubkey.Verify(hash, decodedSignature)
			if r == false {
				output[ERROR_INFO] = "Join Group failed, can not verify signature"
				return c.JSON(http.StatusBadRequest, output)
			}

			b64signPubkey, err := dirks.GetEncodedPubkeyByAlias(SignAlias, localcrypto.Sign)
			if err != nil {
				return rumerrors.NewBadRequestError(rumerrors.ErrGetSignPubKey)
			}
			signPubkey, err := base64.RawURLEncoding.DecodeString(b64signPubkey)
			if err != nil {
				return rumerrors.NewBadRequestError(rumerrors.ErrInvalidSignPubKey)
			}

			encryptPubkey, err := dirks.GetEncodedPubkeyByAlias(EncryptAlias, localcrypto.Encrypt)
			if err != nil {
				return rumerrors.NewBadRequestError("Get encrypt pubkey failed")
			}

			//create nodesdkgroupitem
			item := &quorumpb.NodeSDKGroupItem{}
			group := &quorumpb.GroupItem{}

			group.GroupId = seed.GenesisBlock.GroupId
			group.GroupName = seed.GroupName
			group.OwnerPubKey = seed.GenesisBlock.ProducerPubKey
			group.UserSignPubkey = base64.RawURLEncoding.EncodeToString(signPubkey)
			group.UserEncryptPubkey = encryptPubkey
			group.LastUpdate = 0 //update after getGroupInfo from ChainSDKAPI
			group.HighestHeight = 0
			group.HighestBlockId = ""
			group.GenesisBlock = seed.GenesisBlock

			switch seed.EncryptionType {
			case "private":
				return rumerrors.NewBadRequestError(rumerrors.ErrPrivateGroupNotSupported)
			case "public":
				group.EncryptType = quorumpb.GroupEncryptType_PUBLIC
			default:
				return rumerrors.NewBadRequestError(rumerrors.ErrEncryptionTypeNotSupported)
			}

			switch seed.ConsensusType {
			case "poa":
				group.ConsenseType = quorumpb.GroupConsenseType_POA
			case "pos":
				group.ConsenseType = quorumpb.GroupConsenseType_POS
			default:
				return rumerrors.NewBadRequestError(rumerrors.ErrConsensusTypeNotSupported)
			}

			group.CipherKey = seed.CipherKey
			group.AppKey = seed.AppKey

			item.Group = group
			item.EncryptAlias = EncryptAlias
			item.SignAlias = SignAlias

			if serverurls != nil {
				for _, url := range serverurls {

					if !isValidUrl(url) {
						return rumerrors.NewBadRequestError(rumerrors.ErrInvalidChainAPIURL)
					}
				}
				item.ApiUrl = serverurls
			}
			item.ApiUrl = params.ChainAPIUrl
			seed, err := json.Marshal(params.Seed)
			if err != nil {
				return rumerrors.NewBadRequestError(rumerrors.ErrInvalidChainAPIURL)
			}
			item.GroupSeed = string(seed) //save seed string for future use

			//create joingroup result
			var bufferResult bytes.Buffer
			bufferResult.Write(genesisBlockBytes)
			bufferResult.Write([]byte(group.GroupId))
			bufferResult.Write([]byte(group.GroupName))
			bufferResult.Write(ownerPubkeyBytes)
			bufferResult.Write(signPubkey)
			bufferResult.Write([]byte(encryptPubkey))
			bufferResult.Write([]byte(group.CipherKey))
			hashResult := localcrypto.Hash(bufferResult.Bytes())
			signature, err := ks.SignByKeyAlias(item.SignAlias, hashResult)
			encodedSign := hex.EncodeToString(signature)
			pbGroupSeed := handlers.ToPbGroupSeed(*seed)
			//save seed to db
			if err := nodesdkctx.GetCtx().GetChainStorage().SetGroupSeed(&pbGroupSeed); err != nil {
				output[ERROR_INFO] = fmt.Sprintf("save group seed failed: %s", err)
				return c.JSON(http.StatusBadRequest, output)
			}
			//save nodesdkgroupitem to db
			err = nodesdkctx.GetCtx().GetChainStorage().AddGroupV2(item)
			if err != nil {
				return rumerrors.NewBadRequestError(err)
			}

			joinGrpResult := &JoinGroupResult{
				GroupId:        group.GroupId,
				GroupName:      group.GroupName,
				OwnerPubkey:    group.OwnerPubKey,
				ConsensusType:  seed.ConsensusType,
				EncryptionType: seed.EncryptionType,
				SignAlias:      SignAlias, EncryptAlias: EncryptAlias, CipherKey: group.CipherKey,
				AppKey:    group.AppKey,
				Signature: encodedSign,
			}
			return c.JSON(http.StatusOK, joinGrpResult)
		}
	}
}

// isValidUrl tests a string to determine if it is a well-structured url or not.
func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

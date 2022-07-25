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
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type JoinGroupParamV2 struct {
	Seed         string `json:"seed" validate:"required"` // seed url
	SignAlias    string `json:"sign_alias" validate:"required"`
	EncryptAlias string `json:"encrypt_alias" validate:"required"`
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
		cc := c.(*utils.CustomContext)

		var err error
		payload := new(JoinGroupParamV2)
		if err := cc.BindAndValidate(payload); err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		seed, chainapiUrls, err := handlers.UrlToGroupSeed(payload.Seed)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		signAlias := payload.SignAlias
		encryptAlias := payload.EncryptAlias
		genesisBlockBytes, err := json.Marshal(seed.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError(fmt.Errorf("unmarshal genesis block failed with msg: %s", err))
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			return rumerrors.NewBadRequestError("Open keystore failed")
		}

		signKeyName := dirks.AliasToKeyname(signAlias)
		if signKeyName == "" {
			return rumerrors.NewBadRequestError("sign alias is not exist")
		}

		encryptKeyName := dirks.AliasToKeyname(encryptAlias)
		if encryptKeyName == "" {
			return rumerrors.NewBadRequestError("encrypt alias is not exist")
		}

		//should check keytype

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
		alias, ok := allAlias[encryptAlias]
		if !ok {
			return rumerrors.NewBadRequestError("encrypt alias is not exist")
		}

		if alias.Type != localcrypto.Encrypt {
			return rumerrors.NewBadRequestError("Type mismatch for given encrypt alias")
		}

		alias, ok = allAlias[signAlias]
		if !ok {
			return rumerrors.NewBadRequestError("Sign alias is not exist")
		}

		if alias.Type != localcrypto.Sign {
			return rumerrors.NewBadRequestError("Type mismatch for given sign alias")
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

		r, err := rumchaindata.VerifyBlockSign(seed.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if r == false {
			return rumerrors.NewBadRequestError("Join Group failed, can not verify signature")
		}

		b64signPubkey, err := dirks.GetEncodedPubkeyByAlias(signAlias, localcrypto.Sign)
		if err != nil {
			return rumerrors.NewBadRequestError("Get Sign pubkey failed")
		}

		signPubkey, err := base64.RawURLEncoding.DecodeString(b64signPubkey)
		if err != nil {
			return rumerrors.NewBadRequestError("Decode Sign pubkey failed")
		}

		encryptPubkey, err := dirks.GetEncodedPubkeyByAlias(encryptAlias, localcrypto.Encrypt)
		if err != nil {
			return rumerrors.NewBadRequestError("Get encrypt pubkey failed")
		}

		//create nodesdkgroupitem
		var item *quorumpb.NodeSDKGroupItem
		item = &quorumpb.NodeSDKGroupItem{}

		var group *quorumpb.GroupItem
		group = &quorumpb.GroupItem{}

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
		item.EncryptAlias = encryptAlias
		item.SignAlias = signAlias
		item.ApiUrl = chainapiUrls

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
			return rumerrors.NewBadRequestError(fmt.Errorf("save group seed failed: %s", err))
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
			SignAlias:      signAlias,
			EncryptAlias:   encryptAlias,
			CipherKey:      group.CipherKey,
			AppKey:         group.AppKey,
			Signature:      encodedSign,
		}
		return c.JSON(http.StatusOK, joinGrpResult)
	}
}

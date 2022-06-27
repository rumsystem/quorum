package nodesdkapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GroupSeed struct {
	GenesisBlock   *quorumpb.Block `json:"genesis_block" validate:"required"`
	GroupId        string          `json:"group_id" validate:"required"`
	GroupName      string          `json:"group_name" validate:"required"`
	OwnerPubkey    string          `json:"owner_pubkey" validate:"required"`
	ConsensusType  string          `json:"consensus_type" validate:"required,oneof=pos poa"`
	EncryptionType string          `json:"encryption_type" validate:"required,oneof=public private"`
	CipherKey      string          `json:"cipher_key" validate:"required"`
	AppKey         string          `json:"app_key" validate:"required"`
	Signature      string          `json:"signature" validate:"required"`
}

type JoinGroupParams struct {
	Seed         GroupSeed `json:"seed"          validate:"required"`
	ChainAPIUrl  []string  `json:"urls"          validate:"required"`
	SignAlias    string    `json:"sign_alias"    validate:"required"`
	EncryptAlias string    `json:"encrypt_alias" validate:"required"`
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

func (h *NodeSDKHandler) JoinGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(JoinGroupParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		genesisBlockBytes, err := json.Marshal(params.Seed.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError("unmarshal genesis block failed with msg:" + err.Error())
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			return rumerrors.NewBadRequestError(rumerrors.ErrOpenKeystore.Error())
		}

		signKeyName := dirks.AliasToKeyname(params.SignAlias)
		if signKeyName == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrSignAliasNotFound.Error())
		}

		encryptKeyName := dirks.AliasToKeyname(params.EncryptAlias)
		if encryptKeyName == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrEncryptAliasNotFound.Error())
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
		alias, ok := allAlias[params.EncryptAlias]
		if !ok {
			return rumerrors.NewBadRequestError(rumerrors.ErrEncryptAliasNotFound.Error())
		}

		if alias.Type != localcrypto.Encrypt {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidAliasType.Error())
		}

		alias, ok = allAlias[params.SignAlias]
		if !ok {
			return rumerrors.NewBadRequestError(rumerrors.ErrAliasNotFound.Error())
		}

		if alias.Type != localcrypto.Sign {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidAliasType.Error())
		}

		//check seed signature
		ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.Seed.OwnerPubkey)
		if err != nil {
			return rumerrors.NewBadRequestError("Decode OwnerPubkey failed: " + err.Error())
		}

		ownerPubkey, err := p2pcrypto.UnmarshalPublicKey(ownerPubkeyBytes)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		//decode signature
		decodedSignature, err := hex.DecodeString(params.Seed.Signature)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		//decode cipherkey
		cipherKey, err := hex.DecodeString(params.Seed.CipherKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
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
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		if !verifiy {
			return rumerrors.NewBadRequestError(rumerrors.ErrJoinGroup.Error())
		}

		//*huoju*
		//API does not work???
		//ConfigEncodeKey
		signPubkey, err := dirks.GetEncodedPubkeyByAlias(params.SignAlias, localcrypto.Sign)
		if err != nil {
			return rumerrors.NewBadRequestError(rumerrors.ErrGetSignPubKey.Error())
		}

		pubkeybytes, err := hex.DecodeString(signPubkey)
		if err != nil {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidSignPubKey.Error())
		}

		p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
		decodedsignpubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)

		encryptPubkey, err := dirks.GetEncodedPubkeyByAlias(params.EncryptAlias, localcrypto.Encrypt)
		if err != nil {
			return rumerrors.NewBadRequestError("Get encrypt pubkey failed")
		}

		//create nodesdkgroupitem
		item := &quorumpb.NodeSDKGroupItem{}
		group := &quorumpb.GroupItem{}

		group.GroupId = params.Seed.GroupId
		group.GroupName = params.Seed.GroupName
		group.OwnerPubKey = params.Seed.OwnerPubkey
		group.UserSignPubkey = p2pcrypto.ConfigEncodeKey(decodedsignpubkey)
		group.UserEncryptPubkey = encryptPubkey
		group.LastUpdate = 0 //update after getGroupInfo from ChainSDKAPI
		group.HighestHeight = 0
		group.HighestBlockId = ""
		group.GenesisBlock = params.Seed.GenesisBlock

		switch params.Seed.EncryptionType {
		case "private":
			return rumerrors.NewBadRequestError(rumerrors.ErrPrivateGroupNotSupported.Error())
		case "public":
			group.EncryptType = quorumpb.GroupEncryptType_PUBLIC
		default:
			return rumerrors.NewBadRequestError(rumerrors.ErrEncryptionTypeNotSupported.Error())
		}

		switch params.Seed.ConsensusType {
		case "poa":
			group.ConsenseType = quorumpb.GroupConsenseType_POA
		case "pos":
			group.ConsenseType = quorumpb.GroupConsenseType_POS
		default:
			return rumerrors.NewBadRequestError(rumerrors.ErrConsensusTypeNotSupported.Error())
		}

		group.CipherKey = params.Seed.CipherKey
		group.AppKey = params.Seed.AppKey

		item.Group = group
		item.EncryptAlias = params.EncryptAlias
		item.SignAlias = params.SignAlias

		for _, url := range params.ChainAPIUrl {
			if !isValidUrl(url) {
				return rumerrors.NewBadRequestError(rumerrors.ErrInvalidChainAPIURL.Error())
			}
		}
		item.ApiUrl = params.ChainAPIUrl
		seed, err := json.Marshal(params.Seed)
		if err != nil {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidChainAPIURL.Error())
		}
		item.GroupSeed = string(seed) //save seed string for future use

		//create joingroup result
		var bufferResult bytes.Buffer
		bufferResult.Write(genesisBlockBytes)
		bufferResult.Write([]byte(group.GroupId))
		bufferResult.Write([]byte(group.GroupName))
		bufferResult.Write(ownerPubkeyBytes)
		bufferResult.Write([]byte(signPubkey))
		bufferResult.Write([]byte(encryptPubkey))
		buffer.Write([]byte(params.Seed.ConsensusType))
		buffer.Write([]byte(params.Seed.EncryptionType))
		buffer.Write([]byte(group.CipherKey))
		buffer.Write([]byte(group.AppKey))
		hashResult := localcrypto.Hash(bufferResult.Bytes())
		signature, err := ks.SignByKeyAlias(item.SignAlias, hashResult)
		encodedSign := hex.EncodeToString(signature)

		//save nodesdkgroupitem to db
		err = nodesdkctx.GetCtx().GetChainStorage().AddGroupV2(item)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		joinGrpResult := &JoinGroupResult{
			GroupId:        group.GroupId,
			GroupName:      group.GroupName,
			OwnerPubkey:    group.OwnerPubKey,
			ConsensusType:  params.Seed.ConsensusType,
			EncryptionType: params.Seed.EncryptionType,
			SignAlias:      params.SignAlias,
			EncryptAlias:   params.EncryptAlias,
			CipherKey:      group.CipherKey,
			AppKey:         group.AppKey,
			Signature:      encodedSign,
		}
		return c.JSON(http.StatusOK, joinGrpResult)
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

package nodesdkapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
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
		var err error
		output := make(map[string]string)

		validate := validator.New()
		params := new(JoinGroupParams)
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		genesisBlockBytes, err := json.Marshal(params.Seed.GenesisBlock)
		if err != nil {
			output[ERROR_INFO] = "unmarshal genesis block failed with msg:" + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			output[ERROR_INFO] = "Open keystore failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		signKeyName := dirks.AliasToKeyname(params.SignAlias)
		if signKeyName == "" {
			output[ERROR_INFO] = "sign alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		encryptKeyName := dirks.AliasToKeyname(params.EncryptAlias)
		if encryptKeyName == "" {
			output[ERROR_INFO] = "encrypt alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		//should check keytype

		allKeys, err := dirks.ListAll()
		if err != nil {
			output[ERROR_INFO] = "ListAll failed"
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = "encrypt alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		if alias.Type != localcrypto.Encrypt {
			output[ERROR_INFO] = "Type mismatch for given encrypt alias"
			return c.JSON(http.StatusBadRequest, output)
		}

		alias, ok = allAlias[params.SignAlias]
		if !ok {
			output[ERROR_INFO] = "Sign alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		if alias.Type != localcrypto.Sign {
			output[ERROR_INFO] = "Type mismatch for given sign alias"
			return c.JSON(http.StatusBadRequest, output)
		}

		//check seed signature
		ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.Seed.OwnerPubkey)
		if err != nil {
			output[ERROR_INFO] = "Decode OwnerPubkey failed " + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		ownerPubkey, err := p2pcrypto.UnmarshalPublicKey(ownerPubkeyBytes)
		if err != nil {
			return c.JSON(http.StatusBadRequest, output)
		}

		//decode signature
		decodedSignature, err := hex.DecodeString(params.Seed.Signature)
		if err != nil {
			return c.JSON(http.StatusBadRequest, output)
		}

		//decode cipherkey
		cipherKey, err := hex.DecodeString(params.Seed.CipherKey)
		if err != nil {
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if !verifiy {
			output[ERROR_INFO] = "Join Group failed, can not verify signature"
			return c.JSON(http.StatusBadRequest, output)
		}

		//*huoju*
		//API does not work???
		//ConfigEncodeKey
		signPubkey, err := dirks.GetEncodedPubkeyByAlias(params.SignAlias, localcrypto.Sign)
		if err != nil {
			output[ERROR_INFO] = "Get Sign pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}
		pubkeybytes, err := hex.DecodeString(signPubkey)
		if err != nil {
			output[ERROR_INFO] = "Decode Sign pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
		decodedsignpubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)

		encryptPubkey, err := dirks.GetEncodedPubkeyByAlias(params.EncryptAlias, localcrypto.Encrypt)
		if err != nil {
			output[ERROR_INFO] = "Get encrypt pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		//create nodesdkgroupitem
		var item *quorumpb.NodeSDKGroupItem
		item = &quorumpb.NodeSDKGroupItem{}

		var group *quorumpb.GroupItem
		group = &quorumpb.GroupItem{}

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
			output[ERROR_INFO] = "Private group is not supported by NodeSDK, please use chainsdk to run a full node"
			return c.JSON(http.StatusBadRequest, output)
		case "public":
			group.EncryptType = quorumpb.GroupEncryptType_PUBLIC
		default:
			output[ERROR_INFO] = "Unsupported encryption type"
			return c.JSON(http.StatusBadRequest, output)
		}

		switch params.Seed.ConsensusType {
		case "poa":
			group.ConsenseType = quorumpb.GroupConsenseType_POA
		case "pos":
			group.ConsenseType = quorumpb.GroupConsenseType_POS
		default:
			output[ERROR_INFO] = "Unsupported consensus type"
			return c.JSON(http.StatusBadRequest, output)
		}

		group.CipherKey = params.Seed.CipherKey
		group.AppKey = params.Seed.AppKey

		item.Group = group
		item.EncryptAlias = params.EncryptAlias
		item.SignAlias = params.SignAlias

		for _, url := range params.ChainAPIUrl {
			if !isValidUrl(url) {
				output[ERROR_INFO] = "invalid chainAPI url"
				return c.JSON(http.StatusBadRequest, output)
			}
		}
		item.ApiUrl = params.ChainAPIUrl
		seed, err := json.Marshal(params.Seed)
		if err != nil {
			output[ERROR_INFO] = "invalid chainAPI url"
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		joinGrpResult := &JoinGroupResult{GroupId: group.GroupId, GroupName: group.GroupName, OwnerPubkey: group.OwnerPubKey, ConsensusType: params.Seed.ConsensusType, EncryptionType: params.Seed.EncryptionType, SignAlias: params.SignAlias, EncryptAlias: params.EncryptAlias, CipherKey: group.CipherKey, AppKey: group.AppKey, Signature: encodedSign}
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
		}
		SignAlias := params["sign_alias"]
		EncryptAlias := params["encrypt_alias"]
		genesisBlockBytes, err := json.Marshal(seed.GenesisBlock)
		if err != nil {
			output[ERROR_INFO] = "unmarshal genesis block failed with msg:" + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			output[ERROR_INFO] = "Open keystore failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		signKeyName := dirks.AliasToKeyname(SignAlias)
		if signKeyName == "" {
			output[ERROR_INFO] = "sign alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		encryptKeyName := dirks.AliasToKeyname(EncryptAlias)
		if encryptKeyName == "" {
			output[ERROR_INFO] = "encrypt alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		//should check keytype

		allKeys, err := dirks.ListAll()
		if err != nil {
			output[ERROR_INFO] = "ListAll failed"
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = "encrypt alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		if alias.Type != localcrypto.Encrypt {
			output[ERROR_INFO] = "Type mismatch for given encrypt alias"
			return c.JSON(http.StatusBadRequest, output)
		}

		alias, ok = allAlias[SignAlias]
		if !ok {
			output[ERROR_INFO] = "Sign alias is not exist"
			return c.JSON(http.StatusBadRequest, output)
		}

		if alias.Type != localcrypto.Sign {
			output[ERROR_INFO] = "Type mismatch for given sign alias"
			return c.JSON(http.StatusBadRequest, output)
		}

		//check seed signature
		ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(seed.OwnerPubkey)
		if err != nil {
			output[ERROR_INFO] = "Decode OwnerPubkey failed " + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		//decode signature
		//decodedSignature, err := hex.DecodeString(seed.Signature)
		//if err != nil {
		//	return c.JSON(http.StatusBadRequest, output)
		//}

		//decode cipherkey
		//cipherKey, err := hex.DecodeString(seed.CipherKey)
		//if err != nil {
		//	return c.JSON(http.StatusBadRequest, output)
		//}

		//var buffer bytes.Buffer
		//buffer.Write(genesisBlockBytes)
		//buffer.Write([]byte(params.Seed.GroupId))
		//buffer.Write([]byte(params.Seed.GroupName))
		//buffer.Write(ownerPubkeyBytes)
		//buffer.Write(groupSignPubkey)
		////buffer.Write([]byte(params.Seed.ConsensusType))
		//buffer.Write([]byte(params.Seed.EncryptionType))
		//buffer.Write([]byte(params.Seed.AppKey))
		//buffer.Write(cipherKey)

		//hash := localcrypto.Hash(buffer.Bytes())
		//verifiy, err := ownerPubkey.Verify(hash, decodedSignature)
		//if err != nil {
		//	output[ERROR_INFO] = err.Error()
		//	return c.JSON(http.StatusBadRequest, output)
		//}

		//if !verifiy {
		//	output[ERROR_INFO] = "Join Group failed, can not verify signature"
		//	return c.JSON(http.StatusBadRequest, output)
		//}

		r, err := rumchaindata.VerifyBlockSign(seed.GenesisBlock)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if r == false {
			output[ERROR_INFO] = "Join Group failed, can not verify signature"
			return c.JSON(http.StatusBadRequest, output)
		}

		//*huoju*
		//API does not work???
		//ConfigEncodeKey
		signPubkey, err := dirks.GetEncodedPubkeyByAlias(SignAlias, localcrypto.Sign)
		if err != nil {
			output[ERROR_INFO] = "Get Sign pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}
		pubkeybytes, err := hex.DecodeString(signPubkey)
		if err != nil {
			output[ERROR_INFO] = "Decode Sign pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
		decodedsignpubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)

		encryptPubkey, err := dirks.GetEncodedPubkeyByAlias(EncryptAlias, localcrypto.Encrypt)
		if err != nil {
			output[ERROR_INFO] = "Get encrypt pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		//create nodesdkgroupitem
		var item *quorumpb.NodeSDKGroupItem
		item = &quorumpb.NodeSDKGroupItem{}

		var group *quorumpb.GroupItem
		group = &quorumpb.GroupItem{}

		group.GroupId = seed.GenesisBlock.GroupId
		group.GroupName = seed.GroupName
		group.OwnerPubKey = seed.GenesisBlock.ProducerPubKey
		group.UserSignPubkey = p2pcrypto.ConfigEncodeKey(decodedsignpubkey)
		group.UserEncryptPubkey = encryptPubkey
		group.LastUpdate = 0 //update after getGroupInfo from ChainSDKAPI
		group.HighestHeight = 0
		group.HighestBlockId = ""
		group.GenesisBlock = seed.GenesisBlock

		switch seed.EncryptionType {
		case "private":
			output[ERROR_INFO] = "Private group is not supported by NodeSDK, please use chainsdk to run a full node"
			return c.JSON(http.StatusBadRequest, output)
		case "public":
			group.EncryptType = quorumpb.GroupEncryptType_PUBLIC
		default:
			output[ERROR_INFO] = "Unsupported encryption type"
			return c.JSON(http.StatusBadRequest, output)
		}

		switch seed.ConsensusType {
		case "poa":
			group.ConsenseType = quorumpb.GroupConsenseType_POA
		case "pos":
			group.ConsenseType = quorumpb.GroupConsenseType_POS
		default:
			output[ERROR_INFO] = "Unsupported consensus type"
			return c.JSON(http.StatusBadRequest, output)
		}

		group.CipherKey = seed.CipherKey
		group.AppKey = seed.AppKey

		item.Group = group
		item.EncryptAlias = EncryptAlias
		item.SignAlias = SignAlias

		if serverurls != nil {
			for _, url := range serverurls {
				if !isValidUrl(url) {
					output[ERROR_INFO] = "invalid chainAPI url"
					return c.JSON(http.StatusBadRequest, output)
				}
			}
			item.ApiUrl = serverurls
		}
		jsonseed, err := json.Marshal(seed)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		item.GroupSeed = string(jsonseed) //save seed string for future use

		//create joingroup result
		var bufferResult bytes.Buffer
		bufferResult.Write(genesisBlockBytes)
		bufferResult.Write([]byte(group.GroupId))
		bufferResult.Write([]byte(group.GroupName))
		bufferResult.Write(ownerPubkeyBytes)
		bufferResult.Write(decodedsignpubkey)
		bufferResult.Write([]byte(encryptPubkey))
		bufferResult.Write([]byte(group.CipherKey))
		hashResult := localcrypto.Hash(bufferResult.Bytes())
		signature, err := ks.SignByKeyAlias(item.SignAlias, hashResult)
		encodedSign := hex.EncodeToString(signature)
		//save nodesdkgroupitem to db
		err = nodesdkctx.GetCtx().GetChainStorage().AddGroupV2(item)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		joinGrpResult := &JoinGroupResult{GroupId: group.GroupId, GroupName: group.GroupName, OwnerPubkey: group.OwnerPubKey, ConsensusType: seed.ConsensusType, EncryptionType: seed.EncryptionType, SignAlias: SignAlias, EncryptAlias: EncryptAlias, CipherKey: group.CipherKey, AppKey: group.AppKey, Signature: encodedSign}
		return c.JSON(http.StatusOK, joinGrpResult)
	}
}

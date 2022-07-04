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

		ownerPubkeyBytes, err := base64.RawURLEncoding.DecodeString(seed.GenesisBlock.ProducerPubKey)

		r, err := rumchaindata.VerifyBlockSign(seed.GenesisBlock)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if r == false {
			output[ERROR_INFO] = "Join Group failed, can not verify signature"
			return c.JSON(http.StatusBadRequest, output)
		}

		b64signPubkey, err := dirks.GetEncodedPubkeyByAlias(SignAlias, localcrypto.Sign)
		if err != nil {
			output[ERROR_INFO] = "Get Sign pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}
		signPubkey, err := base64.RawURLEncoding.DecodeString(b64signPubkey)

		if err != nil {
			output[ERROR_INFO] = "Decode Sign pubkey failed"
			return c.JSON(http.StatusBadRequest, output)
		}

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
		group.UserSignPubkey = base64.RawURLEncoding.EncodeToString(signPubkey)
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
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		joinGrpResult := &JoinGroupResult{GroupId: group.GroupId, GroupName: group.GroupName, OwnerPubkey: group.OwnerPubKey, ConsensusType: seed.ConsensusType, EncryptionType: seed.EncryptionType, SignAlias: SignAlias, EncryptAlias: EncryptAlias, CipherKey: group.CipherKey, AppKey: group.AppKey, Signature: encodedSign}
		return c.JSON(http.StatusOK, joinGrpResult)
	}
}

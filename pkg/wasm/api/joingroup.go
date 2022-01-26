//go:build js && wasm
// +build js,wasm

package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

/* from echo handlers, should be refactored later after wasm stabeld */

type JoinGroupResult struct {
	GroupId           string `json:"group_id"`
	GroupName         string `json:"group_name"`
	OwnerPubkey       string `json:"owner_pubkey"`
	UserPubkey        string `json:"user_pubkey"`
	UserEncryptPubkey string `json:"user_encryptpubkey"`
	ConsensusType     string `json:"consensus_type"`
	EncryptionType    string `json:"encryption_type"`
	CipherKey         string `json:"cipher_key"`
	AppKey            string `json:"app_key"`
	Signature         string `json:"signature"`
}

func JoinGroup(paramsBytes []byte) (*JoinGroupResult, error) {
	params := handlers.GroupSeed{}
	err := json.Unmarshal(paramsBytes, &params)
	if err != nil {
		return nil, err
	}

	ks := nodectx.GetNodeCtx().Keystore
	bks, ok := ks.(*localcrypto.BrowserKeystore)
	if !ok {
		return nil, errors.New("Failed to get browser keystore")
	}

	/* Parse some useful bytes */
	ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.OwnerPubkey)
	if err != nil {
		return nil, err
	}
	genesisBlockBytes, err := json.Marshal(params.GenesisBlock)
	if err != nil {
		return nil, err
	}

	/* Verify Seed */
	verify, err := verifySeed(&params, ownerPubkeyBytes, genesisBlockBytes)
	if err != nil {
		return nil, err
	}
	if !verify {
		return nil, errors.New("Failed to verify seed")
	}
	println("Verify Seed: OK")

	/* Load or generate sign/encode key */
	groupSignPubkey, err := initSignKey(params.GroupId, bks)
	if err != nil {
		return nil, err
	}
	userEncryptKey, err := initEncodeKey(params.GroupId, bks)
	if err != nil {
		return nil, err
	}
	println("Load sign key and encode key: OK")

	/* Create GroupItem */
	item := createGroupItem(&params, ownerPubkeyBytes, groupSignPubkey, userEncryptKey)
	println("createGroupItem: OK")

	/* Create Group */
	var group *chain.Group
	group = &chain.Group{}
	err = group.CreateGrp(item)
	if err != nil {
		return nil, err
	}
	err = group.StartSync()
	if err != nil {
		return nil, err
	}
	println("create group: OK")

	/* Add group to context */
	groupmgr := chain.GetGroupMgr()
	groupmgr.Groups[group.Item.GroupId] = group

	/* Sign the result */
	encodedSign, err := signJoinResult(ks, item, genesisBlockBytes, ownerPubkeyBytes, groupSignPubkey, userEncryptKey)
	if err != nil {
		return nil, err
	}
	println("sign result: OK")

	// save group seed to appdata
	pbGroupSeed := handlers.ToPbGroupSeed(params)
	if err := quorumContext.GetWASMContext().AppDb.SetGroupSeed(&pbGroupSeed); err != nil {

	}

	ret := JoinGroupResult{GroupId: item.GroupId, GroupName: item.GroupName, OwnerPubkey: item.OwnerPubKey, ConsensusType: params.ConsensusType, EncryptionType: params.EncryptionType, UserPubkey: item.UserSignPubkey, UserEncryptPubkey: userEncryptKey, CipherKey: item.CipherKey, AppKey: item.AppKey, Signature: encodedSign}

	return &ret, nil
}

func verifySeed(params *handlers.GroupSeed, ownerPubkeyBytes []byte, genesisBlockBytes []byte) (bool, error) {
	verify := false
	decodedSignature, err := hex.DecodeString(params.Signature)
	if err != nil {
		return verify, err
	}

	ownerPubkey, err := p2pcrypto.UnmarshalPublicKey(ownerPubkeyBytes)
	if err != nil {
		return verify, err
	}
	cipherKey, err := hex.DecodeString(params.CipherKey)
	if err != nil {
		return verify, err
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
	return ownerPubkey.Verify(hash, decodedSignature)
}

func initSignKey(groupId string, bks *localcrypto.BrowserKeystore) ([]byte, error) {
	hexKey, err := bks.GetEncodedPubkey(groupId, localcrypto.Sign)
	if err != nil && strings.HasPrefix(err.Error(), "key not exist") {
		/* Create one */
		newSignAddr, err := bks.NewKey(groupId, localcrypto.Sign, "")
		if err == nil && newSignAddr != "" {
			hexKey, err = bks.GetEncodedPubkey(groupId, localcrypto.Sign)
		} else {
			return nil, err
		}
	}

	pubKeyBytes, err := hex.DecodeString(hexKey)
	p2pPubKey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubKeyBytes)
	return p2pcrypto.MarshalPublicKey(p2pPubKey)
}

func initEncodeKey(groupId string, bks *localcrypto.BrowserKeystore) (string, error) {
	userEncryptKey, err := bks.GetEncodedPubkey(groupId, localcrypto.Encrypt)
	if err != nil {
		if strings.HasPrefix(err.Error(), "key not exist") {
			userEncryptKey, err = bks.NewKey(groupId, localcrypto.Encrypt, "")
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return userEncryptKey, nil
}

func createGroupItem(params *handlers.GroupSeed, ownerPubkeyBytes []byte, groupSignPubkey []byte, userEncryptKey string) *quorumpb.GroupItem {
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

	item.UserEncryptPubkey = userEncryptKey
	item.UserSignPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)

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

	return item
}

func signJoinResult(ks localcrypto.Keystore, item *quorumpb.GroupItem, genesisBlockBytes []byte, ownerPubkeyBytes []byte, groupSignPubkey []byte, userEncryptKey string) (string, error) {
	var bufferResult bytes.Buffer
	bufferResult.Write(genesisBlockBytes)
	bufferResult.Write([]byte(item.GroupId))
	bufferResult.Write([]byte(item.GroupName))
	bufferResult.Write(ownerPubkeyBytes)
	bufferResult.Write(groupSignPubkey)
	bufferResult.Write([]byte(userEncryptKey))

	hashResult := chain.Hash(bufferResult.Bytes())
	signature, err := ks.SignByKeyName(item.GroupId, hashResult)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(signature), nil
}

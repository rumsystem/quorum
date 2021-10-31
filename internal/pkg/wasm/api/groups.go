//go:build js && wasm
// +build js,wasm

package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

/* from echo handlers, should be refactored later after wasm stabeld */

type JoinGroupParam struct {
	GenesisBlock   *quorumpb.Block `from:"genesis_block" json:"genesis_block" validate:"required"`
	GroupId        string          `from:"group_id" json:"group_id" validate:"required"`
	GroupName      string          `from:"group_name" json:"group_name" validate:"required"`
	OwnerPubKey    string          `from:"owner_pubkey" json:"owner_pubkey" validate:"required"`
	ConsensusType  string          `from:"consensus_type" json:"consensus_type" validate:"required"`
	EncryptionType string          `from:"encryption_type" json:"encryption_type" validate:"required"`
	CipherKey      string          `from:"cipher_key" json:"cipher_key" validate:"required"`
	AppKey         string          `from:"app_key" json:"app_key" validate:"required"`
	Signature      string          `from:"signature" json:"signature" validate:"required"`
}

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
	params := JoinGroupParam{}
	ret := JoinGroupResult{}
	err := json.Unmarshal(paramsBytes, &params)
	if err != nil {
		return nil, err
	}

	verify, err := verifySeed(&params)
	if err != nil {
		return nil, err
	}
	if !verify {
		return nil, errors.New("Failed to verify seed")
	}

	println("Verify Seed: OK")

	return &ret, nil
}

func verifySeed(params *JoinGroupParam) (bool, error) {
	verify := false
	decodedSignature, err := hex.DecodeString(params.Signature)
	if err != nil {
		return verify, err
	}
	genesisBlockBytes, err := json.Marshal(params.GenesisBlock)
	if err != nil {
		return verify, err
	}
	ownerPubkeyBytes, err := p2pcrypto.ConfigDecodeKey(params.OwnerPubKey)
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

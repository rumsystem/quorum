package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

type ClearGroupDataParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

type ClearGroupDataResult struct {
	GroupId   string `json:"group_id" validate:"required"`
	Signature string `json:"signature" validate:"required"`
}

func ClearGroupData(params *ClearGroupDataParam) (*ClearGroupDataResult, error) {

	validate := validator.New()
	err := validate.Struct(params)
	if err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return nil, fmt.Errorf("Group %s not exist", params.GroupId)
	}

	// stop syncing first, to avoid starving in browser (indexeddb)
	if err := group.StopSync(); err != nil {
		return nil, err
	}

	if err := group.ClearGroup(); err != nil {
		return nil, err
	}

	var groupSignPubkey []byte
	ks := localcrypto.GetKeystore()
	hexkey, err := ks.GetEncodedPubkey("default", localcrypto.Sign)
	pubkeybytes, err := hex.DecodeString(hexkey)
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	var buffer bytes.Buffer
	buffer.Write(groupSignPubkey)
	buffer.Write([]byte(params.GroupId))
	hash := chain.Hash(buffer.Bytes())
	signature, err := ks.SignByKeyName(params.GroupId, hash)
	encodedString := hex.EncodeToString(signature)

	return &ClearGroupDataResult{GroupId: params.GroupId, Signature: encodedString}, nil
}

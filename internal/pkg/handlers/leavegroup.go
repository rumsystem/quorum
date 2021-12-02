package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

type LeaveGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

type LeaveGroupResult struct {
	GroupId   string `json:"group_id" validate:"required"`
	Signature string `json:"signature" validate:"required"`
}

func LeaveGroup(params *LeaveGroupParam, appdb *appdata.AppDb) (*LeaveGroupResult, error) {

	validate := validator.New()

	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return nil, fmt.Errorf("Group %s not exist", params.GroupId)
	}

	if err := group.LeaveGrp(); err != nil {
		return nil, err
	}

	delete(groupmgr.Groups, params.GroupId)

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

	// delete group seed from appdata
	if err := appdb.DelGroupSeed(params.GroupId); err != nil {
		return nil, fmt.Errorf("save group seed failed: %s", err)
	}

	return &LeaveGroupResult{GroupId: params.GroupId, Signature: encodedString}, nil
}

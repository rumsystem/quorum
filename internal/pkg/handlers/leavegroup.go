package handlers

import (
	"bytes"
	"encoding/hex"
	"fmt"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
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

func LeaveGroup(params *LeaveGroupParam) (*LeaveGroupResult, error) {
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; ok {
		err := group.LeaveGrp()

		if err != nil {
			return nil, err
		}

		delete(groupmgr.Groups, params.GroupId)
		if err != nil {
			return nil, err
		}

		var groupSignPubkey []byte
		ks := localcrypto.GetKeystore()
		hexkey, err := ks.GetEncodedPubkey("default", localcrypto.Sign)
		pubkeybytes, err := hex.DecodeString(hexkey)
		p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
		groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
		if err != nil {
			return nil, fmt.Errorf("group key can't be decoded, err: %s", err.Error())
		}

		var buffer bytes.Buffer
		buffer.Write(groupSignPubkey)
		buffer.Write([]byte(params.GroupId))
		hash := chain.Hash(buffer.Bytes())
		signature, err := ks.SignByKeyName(params.GroupId, hash)
		encodedString := hex.EncodeToString(signature)

		return &LeaveGroupResult{GroupId: params.GroupId, Signature: encodedString}, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", params.GroupId)
	}
}

package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type DenyListParam struct {
	Action  string `from:"action" json:"action" validate:"required,oneof=add del"`
	PeerId  string `from:"peer_id"      json:"peer_id"      validate:"required"`
	GroupId string `from:"group_id"  json:"group_id"  validate:"required"`
	Memo    string `from:"memo"  json:"memo"  `
}

type DenyUserResult struct {
	GroupId          string `json:"group_id" validate:"required"`
	PeerId           string `json:"peer_id" validate:"required"`
	GroupOwnerPubkey string `json:"owner_pubkey" validate:"required"`
	Sign             string `json:"sign" validate:"required"`
	TrxId            string `json:"trx_id" validate:"required"`
	Action           string `json:"action" validate:"required"`
	Memo             string `json:"memo" validate:"required"`
}

func MgrGrpBlkList(params *DenyListParam) (*DenyUserResult, error) {

	validate := validator.New()

	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	ks := nodectx.GetNodeCtx().Keystore

	hexkey, err := ks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
	pubkeybytes, err := hex.DecodeString(hexkey)
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	item := &quorumpb.DenyUserItem{}
	item.GroupId = params.GroupId
	item.PeerId = params.PeerId
	item.GroupOwnerPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	item.Action = params.Action
	item.Memo = params.Memo

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		return nil, errors.New("Can not find group")
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return nil, errors.New("Only group owner can add or remove user to blocklist")
	} else {
		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.PeerId))
		buffer.Write(groupSignPubkey)
		buffer.Write([]byte(item.Action))
		buffer.Write([]byte(item.Memo))
		hash := chain.Hash(buffer.Bytes())

		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			return nil, err
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdBlkList(item)

		if err != nil {
			return nil, err
		}

		blockGrpUserResult := &DenyUserResult{GroupId: item.GroupId, PeerId: item.PeerId, GroupOwnerPubkey: p2pcrypto.ConfigEncodeKey(groupSignPubkey), Sign: hex.EncodeToString(signature), Action: item.Action, Memo: item.Memo, TrxId: trxId}

		return blockGrpUserResult, nil
	}
}

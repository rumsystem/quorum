package handlers

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type LeaveGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

type LeaveGroupResult struct {
	GroupId string `json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
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

	group.StopSync()
	if err := group.LeaveGrp(); err != nil {
		return nil, err
	}

	delete(groupmgr.Groups, params.GroupId)

	//var groupSignPubkey []byte
	//ks := localcrypto.GetKeystore()

	//hexkey, err := ks.GetEncodedPubkey("default", localcrypto.Sign)
	//pubkeybytes, err := hex.DecodeString(hexkey)
	//p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	//groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
	//if err != nil {
	//	return nil, errors.New("group key can't be decoded, err:" + err.Error())
	//}

	//var buffer bytes.Buffer
	//buffer.Write(groupSignPubkey)
	//buffer.Write([]byte(params.GroupId))
	//hash := localcrypto.Hash(buffer.Bytes())
	//signature, err := ks.EthSignByKeyName(params.GroupId, hash)
	//encodedString := hex.EncodeToString(signature)

	// delete group seed from appdata
	if err := appdb.DelGroupSeed(params.GroupId); err != nil {
		return nil, fmt.Errorf("delete group seed failed: %s", err)
	}

	return &LeaveGroupResult{GroupId: params.GroupId}, nil
}

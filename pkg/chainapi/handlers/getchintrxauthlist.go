package handlers

import (
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type TrxAuthItem struct {
	TrxType  string `example:"POST"`
	AuthType string `example:"FOLLOW_ALW_LIST"`
}

type TrxAuthParams struct {
	GroupId string `param:"group_id" validate:"required,uuid4" example:"b3e1800a-af6e-4c67-af89-4ddcf831b6f7"`
	TrxType string `param:"trx_type" validate:"required,oneof=POST ANNOUNCE REQ_BLOCK" example:"POST"`
}

func GetChainTrxAuthMode(chainapidb def.APIHandlerIface, groupid string, trxType string) (*TrxAuthItem, error) {
	trxAuthItem := TrxAuthItem{}
	if groupid == "" {
		return nil, rumerrors.ErrInvalidGroupID
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[groupid]
	if !ok {
		return nil, rumerrors.ErrGroupNotFound
	}

	trxTypeProto, err := getTrxTypeByString(trxType)
	if err != nil {
		return nil, err
	}

	trxAuthType, err := chainapidb.GetTrxAuthModeByGroupId(group.GroupId, trxTypeProto, group.Nodename)
	if err != nil {
		return nil, err
	}

	trxAuthItem.TrxType = trxTypeProto.String()
	trxAuthItem.AuthType = trxAuthType.String()
	return &trxAuthItem, nil
}

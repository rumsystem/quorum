package handlers

import (
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type TrxAuthItem struct {
	TrxType  string
	AuthType string
}

type TrxAuthParams struct {
	GroupId string `param:"group_id" validate:"required,uuid4"`
	TrxType string `param:"trx_type" validate:"required,oneof=POST ANNOUNCE REQ_BLOCK_FORWARD REQ_BLOCK_BACKWARD BLOCK_SYNCED BLOCK_PRODUCED ASK_PEERID"`
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

	trxAuthType, err := chainapidb.GetTrxAuthModeByGroupId(group.Item.GroupId, trxTypeProto, group.ChainCtx.GetNodeName())
	if err != nil {
		return nil, err
	}

	trxAuthItem.TrxType = trxTypeProto.String()
	trxAuthItem.AuthType = trxAuthType.String()
	return &trxAuthItem, nil
}

package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type TrxAuthItem struct {
	TrxType  string
	AuthType string
}

func GetChainTrxAuthMode(groupid string, trxType string) (*TrxAuthItem, error) {
	trxAuthItem := TrxAuthItem{}

	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {

		trxTypeProto, err := getTrxTypeByString(trxType)
		if err != nil {
			return nil, err
		}

		trxAuthType, err := group.GetSendTrxAuthMode(trxTypeProto)
		if err != nil {
			return nil, err
		}
		trxAuthItem.TrxType = trxTypeProto.String()
		trxAuthItem.AuthType = trxAuthType.String()
		return &trxAuthItem, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}

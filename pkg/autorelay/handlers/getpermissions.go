package handlers

import (
	"strconv"

	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type GetPermissionsResult struct {
	AllowReserve bool
	AllowConnect bool
}

func GetPermissions(db storage.QuorumStorage, peer string) (*GetPermissionsResult, error) {
	res := &GetPermissionsResult{true, true}

	// only get `AllowConnect` permission here, we always allow reserve for now
	k := []byte(GetAllowConnectKey(peer))

	isExist, err := db.IsExist(k)
	if err != nil {
		return nil, err
	}
	if !isExist {
		// allow by default
		return res, nil
	}

	v, err := db.Get(k)
	if err != nil {
		return nil, err
	}
	ok, _ := strconv.ParseBool(string(v))
	res.AllowConnect = ok

	return res, nil
}

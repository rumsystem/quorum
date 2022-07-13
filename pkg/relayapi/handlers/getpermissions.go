package handlers

import (
	"fmt"
	"strconv"

	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type GetPermissionsResult struct {
	AllowReserve bool
	AllowConnect bool
}

func GetPermissions(db storage.QuorumStorage, peer string) (*GetPermissionsResult, error) {
	res := &GetPermissionsResult{true, true}

	k := []byte(fmt.Sprintf("%s_%s", PREFIX_ALLOW_CONNECT, peer))

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

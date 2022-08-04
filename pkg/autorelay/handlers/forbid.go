package handlers

import (
	"strconv"

	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type ForbidParam struct {
	Peer string `json:"peer"`
}

type ForbidResult struct {
	Ok bool `json:"ok"`
}

/* ForbidPeer forbid a server peer */
func ForbidPeer(db storage.QuorumStorage, param ForbidParam) (*ForbidResult, error) {
	res := &ForbidResult{true}

	k := []byte(GetAllowConnectKey(param.Peer))
	if err := db.Set(k, []byte(strconv.FormatBool(false))); err != nil {
		return nil, err
	}

	return res, nil
}

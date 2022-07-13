package handlers

import "github.com/rumsystem/quorum/internal/pkg/storage"

type ForbidParam struct {
	Peer string `json:"peer"`
}

type ForbidResult struct {
	Ok bool `json:"ok"`
}

func ForbidPeer(db storage.QuorumStorage, param ForbidParam) (ForbidResult, error) {
	res := ForbidResult{true}

	return res, nil
}

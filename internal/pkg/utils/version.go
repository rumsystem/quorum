package utils

import (
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

var GitCommit string

func SetGitCommit(hash string) {
	GitCommit = hash
}

func GetQuorumVersion() string {
	return nodectx.GetNodeCtx().Version + " - " + GitCommit
}

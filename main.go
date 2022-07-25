package main

import (
	"github.com/rumsystem/quorum/cmd"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

var (
	ReleaseVersion string
	GitCommit      string
)

// @title Quorum Api
// @version 1.0
// @description Quorum Api Docs
// @BasePath /
func main() {
	if ReleaseVersion == "" {
		ReleaseVersion = "v1.0.0"
	}
	if GitCommit == "" {
		GitCommit = "devel"
	}
	utils.SetGitCommit(GitCommit)
	utils.SetVersion(ReleaseVersion)

	cmd.Execute()
}

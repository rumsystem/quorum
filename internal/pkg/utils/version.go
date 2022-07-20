package utils

var ReleaseVersion string
var GitCommit string

func SetGitCommit(hash string) {
	GitCommit = hash
}

func SetVersion(version string) {
	ReleaseVersion = version
}

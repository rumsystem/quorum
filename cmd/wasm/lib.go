//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorum "github.com/rumsystem/quorum/pkg/wasm"
)

func main() {
	c := make(chan struct{}, 0)

	println("WASM Go Initialized.\nGit Version: ", utils.GitCommit)

	quorum.RegisterJSFunctions()

	<-c
}

//go:build js && wasm
// +build js,wasm

package main

import (
	quorum "github.com/rumsystem/quorum/pkg/wasm"
)

func main() {
	c := make(chan struct{}, 0)

	println("WASM Go Initialized")

	quorum.RegisterJSFunctions()

	<-c
}

//go:build js && wasm
// +build js,wasm

package api

import "github.com/rumsystem/quorum/pkg/chainapi/handlers"

func AddRelayServers(peers []string) (bool, error) {
	return handlers.AddRelayServers(peers)
}

//go:build js && wasm
// +build js,wasm

package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

func AddRelayServers(peers []string) (bool, error) {
	return handlers.AddRelayServers(peers)
}

//go:build js && wasm
// +build js,wasm

package crypto

import quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"

type BrowserKeystore struct {
	store    *quorumStorage.QSIndexDB
	password string
}

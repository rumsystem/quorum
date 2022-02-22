//go:build !js
// +build !js

package logging

import log "github.com/ipfs/go-log/v2"

func Logger(system string) log.StandardLogger {
	return log.Logger(system)
}

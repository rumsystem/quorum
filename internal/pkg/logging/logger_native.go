//go:build !js
// +build !js

package logging

import (
	log "github.com/ipfs/go-log/v2"
)

func Logger(system string) QuorumLogger {
	return log.Logger(system)
}

func SetLogLevel(name, level string) error {
	return log.SetLogLevel(name, level)
}

func SetAllLoggers(lvl int) {
	log.SetAllLoggers(log.LogLevel(lvl))
}

func LevelFromString(level string) (int, error) {
	l, e := log.LevelFromString(level)
	return int(l), e
}

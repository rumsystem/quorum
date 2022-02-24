package logging

import log "github.com/ipfs/go-log/v2"

type QuorumLogger interface {
	log.StandardLogger
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

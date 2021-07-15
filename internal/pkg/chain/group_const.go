package chain

import logging "github.com/ipfs/go-log/v2"

var group_log = logging.Logger("group")

type GroupStatus int8

const (
	GROUP_CLEAN = 0
	GROUP_DIRTY = 1
)

var WAIT_BLOCK_TIME_S = 10

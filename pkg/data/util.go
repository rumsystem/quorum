package data

import (
	"encoding/binary"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

const POST_OBJ_SIZE_LIMIT = 300 * 1024 //300Kb
var dataLog = logging.Logger("data")

func IsTrxWithinSizeLimit(content []byte) (bool, error) {
	size := binary.Size(content)
	if size > POST_OBJ_SIZE_LIMIT {
		e := fmt.Errorf("content size %dkb over %dkb", size/1024, POST_OBJ_SIZE_LIMIT/1024)
		dataLog.Warn(e)
		return false, e
	}

	return true, nil
}
